import asyncio
import queue

import discord
from async_timeout import timeout
from youtube_dl import DownloadError

from utils.YTDLSource import YTDLSource
from utils.YoutubeApi import *


##########################################################################################

class MusicPlayer:
    __slots__ = (
        'bot', '_guild', '_channel', '_cog', 'queue', 'next', 'current', 'np', 'volume', 'autoplay', 'ctx', 'history', 'skip')

    def __init__(self, ctx):
        self.ctx = ctx
        self.bot = ctx.bot
        self._guild = ctx.guild
        self._channel = ctx.channel
        self._cog = ctx.cog

        self.queue = asyncio.Queue()
        self.next = asyncio.Event()

        self.np = None
        self.volume = .5
        self.current = None
        self.autoplay = False
        self.skip = False
        self.history = []

        ctx.bot.loop.create_task(self.player_loop())

    async def player_loop(self):
        await self.bot.wait_until_ready()

        while not self.bot.is_closed():
            self.next.clear()

            try:
                async with timeout(300):
                    source = await self.queue.get()
            except asyncio.TimeoutError:
                return self.destroy(self._guild)

            if not isinstance(source, YTDLSource):
                try:
                    source = await YTDLSource.regather_stream(source, loop=self.bot.loop)
                except Exception as e:
                    await self._channel.send(f'Fehler beim verarbeiten des Songs.\n'
                                             f'```css\n[{e}\n```')
                    continue

                source.volume = self.volume
                self.current = source
                self.history.append(source.title)

                self._guild.voice_client.play(source, after=lambda _: print(self.bot.loop.call_soon_threadsafe(self.next.set)))
                self.np = await self._channel.send(f'**Now Playing:** `{source.title}` requested by '
                                                   f'`{source.requester}`')

                await self.next.wait()  # Stop right fucking here

                if self.queue.empty():
                    if self.autoplay:
                        relatedVideoUrlList = getRelatedVideoUrlList(source, 3)
                        Success = False
                        relatedSource = ''

                        i = 0
                        while not Success:
                            try:
                                Success = True
                                relatedSource = await YTDLSource.create_source(self.ctx, relatedVideoUrlList[i],
                                                                               loop=self.bot.loop,
                                                                               download=False)

                                for title in self.history:
                                    if relatedSource["title"] == title:
                                        Success = False
                            except DownloadError:
                                Success = False
                                i += 1

                        if relatedSource:
                            await self.queue.put(relatedSource)

                source.cleanup()
                self.current = None
                self.skip = False

                try:
                    await self.np.delete()
                except discord.HTTPException:
                    pass

    def destroy(self, guild):
        return self.bot.loop.create_Task(self._cog.cleanup(guild))
