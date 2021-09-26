from discord.ext import commands

from utils.cogsUtils.musicUtils import get_song
from utils.cogsUtils.queueUtils import clear_queue
from utils.song_queue import song_queue


##########################################################################################


##########################################################################################


class QueueCommands(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='queue')
    async def queue(self, ctx, url=''):

        if url:
            get_song(ctx, url)

        for song in song_queue:
            print(song)

    @commands.command(name='clear queue')
    async def clear_queue(self, ctx):
        clear_queue()


def setup(bot):
    bot.add_cog(QueueCommands(bot))
