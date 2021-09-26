from cogs.botCommands import *

from utils.cogsUtils.musicUtils import *
from utils.cogsUtils.queueUtils import queue_is_empty

##########################################################################################

directory_queued = 'queued/'
directory_playing = 'playing/'
directory_played = 'played/'


##########################################################################################

class PlayCommand(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='play')
    async def play(self, ctx, url=''):
        await join_channel(self, ctx)
        voice_client = self.bot.voice_clients[0]

        if not music_is_playing(voice_client):
            if queue_is_empty():
                get_song(ctx, url)
                play_song(ctx, voice_client)

            elif music_is_paused(voice_client):
                resume_song(voice_client)

            elif not url:
                try:
                    play_song(ctx, voice_client)
                except Exception:
                    await ctx.send(f'no song to play')

##########################################################################################

    @commands.command(name='stop')
    async def stop(self, ctx):
        voice_Client = self.bot.voice_clients[0]
        voice_Client.stop()

        print('MUSIC STOPPED')


##########################################################################################

def setup(bot):
    bot.add_cog(PlayCommand(bot))
