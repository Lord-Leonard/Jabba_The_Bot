from discord.ext import commands
from utils.cogsUtils.musicUtils import *
from song_queue import *
from utils.cogsUtils.osUtils import *
from utils.cogsUtils.musicUtils import *

directory_queued = 'queued/'
directory_playing = 'playing/'


class QueueCommands(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='list-queue')
    async def list_queue(self, ctx):
        for song in Queue.song_queue:
            print(song)

    @commands.command(name='queue')
    async def queue(self, ctx, url):
        song_object = create_song_object(ctx, url)
        download_file(url)
        move_song(song_object, directory_queued)

        add_song_to_queue(song_object)


def setup(bot):
    bot.add_cog(QueueCommands(bot))
