import discord

import asyncio
import itertools
import sys
import traceback

from utils.MusicPlayer import MusicPlayer
from utils.YTDLSource import YTDLSource
from utils.CustomExceptions import *

##########################################################################################

ffmpegopts = {
    'before_options': '-nostdin',
    'options': '-vn'
}


##########################################################################################

class Music(commands.Cog):
    __slots__ = ('bot', 'players')

    def __init__(self, bot):
        self.bot = bot
        self.players = {}

    async def cleanup(self, guild):
        try:
            await guild.voice_client.disconnect()
        except AttributeError:
            pass

        try:
            del self.players[guild.id]
        except KeyError:
            pass

    async def __local_check(self, ctx):
        if not ctx.guild:
            raise commands.NoPrivateMessage
        return True

    async def __error(self, ctx, error):
        if isinstance(error, commands.NoPrivateMessage):
            try:
                return await ctx.send('Nix gehen in privat')
            except discord.HTTPException:
                pass
        elif isinstance(error, InvalidVoiceChannel):
            await ctx.send('Error connecting to Voice Channel.'
                           'Please make sure you are in a valid channel or provide me with one')

        print('Ignoring exception in command {}:'.format(ctx.command), file=sys.stderr)
        traceback.print_exception(type(error), error, error.__traceback__, file=sys.stderr)

    def get_player(self, ctx):
        try:
            player = self.players[ctx.guild.id]
        except KeyError:
            player = MusicPlayer(ctx)
            self.players[ctx.guild.id] = player

        return player

    ##########################################################################################

    @commands.command(name='connect', aliases=['join', 'komm ran'])
    async def connect_(self, ctx, *, channel: discord.VoiceChannel = None):

        if not channel:
            try:
                channel = ctx.author.voice.channel
            except AttributeError:
                raise InvalidVoiceChannel('No channel to join. Please either specify a valid channel or join one.')

        vc = ctx.voice_client

        if vc:
            if vc.channel.id == channel.id:
                return
            try:
                await vc.move_to(channel)
            except asyncio.TimeoutError:
                raise VoiceConnectionError(f'Moving to channel: <{channel}> hat zu lange gedauert')
        else:
            try:
                await channel.connect()
            except asyncio.TimeoutError:
                raise VoiceConnectionError(f'Beitreten des Channels: <{channel}> hat zu lange gedauert')

            await ctx.send(f'Verbindung zu **{channel}** erfolgreich aufgebaut', delete_after=20)

    @commands.command(name='play', aliases=['sing', 'mach', 'spiel'])
    async def play_(self, ctx, *, search: str):
        await ctx.trigger_typing()
        vc = ctx.voice_client

        if not vc:
            await ctx.invoke(self.connect_)
        elif not vc.is_playing():
            return await ctx.send('Musik wird schon abgespielt.', delete_after=20)

        player = self.get_player(ctx)

        source = await YTDLSource.create_source(ctx, search, loop=self.bot.loop, download=False)

        await player.queue.put(source)

    @commands.command(name='pause')
    async def pause_(self, ctx):
        vc = ctx.voice_client

        if not vc or not vc.is_playing():
            return await ctx.send('Es wird gerade keine Musik abgespielt!', delete_after=20)
        elif vc.is_paused():
            return await ctx.send('Musik ist bereits pausiert', delete_after=20)

        vc.pause()
        await ctx.send(f'**`{ctx.author}`** hat die Musik pausiert')

    @commands.command(name='resume')
    async def res_(self, ctx):
        vc = ctx.voice_client

        if not vc or not vc.is_connected():
            return await ctx.send('Gibt nix zum fortsetzten.', delete_after=20)
        elif vc.is_playing():
            return await ctx.send('Musik wird schon abgespielt.', delete_after=20)

        vc.resume()
        await ctx.send(f'**`{ctx.author}`** hat die Musik fortgesetzt.')

    @commands.command(name='skip')
    async def skip_(self, ctx):
        vc = ctx.voice_client

        if not vc or not vc.is_connected():
            return await ctx.send(f'Gibt nix zum skippen.', delete_after=20)

        if vc.is_paused():
            pass
        elif not vc.is_playing():
            return await ctx.send(f'Wat soll ich skippen?!')

        vc.stop()
        await ctx.send(f'**`{ctx.author}`** hat das musikalische Meisterwerk geskippt!')

    @commands.command(name='queue', aliases=['q', 'playlist', 'warteschlange', 'que'])
    async def queue_info(self, ctx, search=''):
        vc = ctx.voice_client

        if not vc or not vc.is_connected():
            return await ctx.send(f'Ich nix connected', delete_after=20)

        player = self.get_player(ctx)

        if not search:
            source = await YTDLSource.create_source(ctx, search, loop=self.bot.loop, download=False)
            await player.queue.put(source)
            return await ctx.send(f'`{source.title}` wurde von `{ctx.author}``in die queue gepackt.')

        if player.queue.empty():
            return await ctx.send('q, welche q?')

        upcomming = list(itertools.islice(player.queue._queue, 0, 5))

        fmt = '\n'.join(f'**``{_["title"]}`**' for _ in upcomming)
        embed = discord.Embed(title=f'Upcommming - Next{len(upcomming)}', description=fmt)

        await ctx.send(embed=embed)

    @commands.command(name='now_playing', aliases=['np', 'current', 'currentsong', 'playing'])
    async def now_playing_(self, ctx):
        vc = ctx.voice_client

        if not vc or not vc.is_connected():
            return await ctx.send(f'Ich nix connected', delete_after=20)

        player = self.get_player(ctx)
        if not player.current:
            return await ctx.send('Ich spiele gerade nix!')

        try:
            await player.np.delete()
        except discord.HTTPException:
            pass

        player.np = await ctx.send(f'`{vc.source.title}` '
                                   f'requested by: `{vc.source.requester}`')

    @commands.command(name='volume', aliases=['vol', 'v', 'lautstärke'])
    async def volume_(self, ctx, *, vol: float):

        vc = ctx.voice_client

        if not vc or not vc.is_connected():
            return await ctx.send(f'Ich nix connected', delete_after=20)

        if not 0 < vol < 101:
            return await ctx.send(f'Gib lautstärke zwischen 0 und 100. Danke')

        player = self.get_player(ctx)

        if vc.source:
            vc.source.volume = vol / 100

        player.volume = vol / 100
        await ctx.send(f'**`{ctx.author}`** hat die Lautstärke auf **{vol}** gesetzt')

    @commands.command(name='stop', aliases=['quit'])
    async def stop_(self, ctx):
        vc = ctx.voice_client

        if not vc or not vc.is_connected():
            return await ctx.sned(f'Ich nix spielen Musik')

        await self.cleanup(ctx.guild)


##########################################################################################

def setup(bot):
    bot.add_cog(Music(bot))
