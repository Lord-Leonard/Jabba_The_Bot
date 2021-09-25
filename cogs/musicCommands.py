from discord.ext import commands
from utils.cogsUtils.musicUtils import *
from utils.cogsUtils.osUtils import *
from cogs.botCommands import *

from utils.cogsUtils.musicUtils import *

directory_queued = 'queued'
directory_playing = 'playing'


class PlayCommand(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='play')
    async def play(self, ctx, url):

        await join_channel(self, ctx)

        voice_client = self.bot.voice_clients[0]
        song_object = create_song_object(ctx, url)

        if not voice_client.is_playing():
            if os.listdir('playing'):
                clear_directory('playing')

            download_file(url)
            move_song(song_object, directory_playing)
            add_song_to_queue(song_object)
            play_song(voice_client, ctx)

            print(f'MUSIC PLAYING')

        else:
            await ctx.send('Music already playing')

################################################################################################

    @commands.command(name='stop')
    async def stop(self, ctx):

        voice_Client = self.bot.voice_clients[0]
        voice_Client.stop()

        print('MUSIC STOPPED')


################################################################################################

def setup(bot):
    bot.add_cog(PlayCommand(bot))
