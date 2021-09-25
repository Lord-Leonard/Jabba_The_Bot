import os
import discord
from discord.ext import commands
from utils.cogsUtils.musicUtils import *
from utils.cogsUtils.osUtils import *
from utils.botUtils.botUtils import *
import youtube_dl


class BotCommands(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='leave')
    async def leave(self, ctx):

        voice_Client = self.bot.voice_clients[0]

        await voice_Client.disconnect()
        print('DISCONNECTED')

###########################################################################################

    @commands.command(name='join')
    async def join(self, ctx):
        await join_channel(self, ctx)


def setup(bot):
    bot.add_cog(BotCommands(bot))
