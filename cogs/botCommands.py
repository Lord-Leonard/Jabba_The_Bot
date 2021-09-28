from discord.ext import commands
from utils.botUtils.botUtils import join_channel, leave_channel


##########################################################################################

class BotCommands(commands.Cog):
    def __init__(self, bot):
        self.bot = bot

    @commands.command(name='leave')
    async def leave(self, ctx):
        await leave_channel(self, ctx)
        print('DISCONNECTED')

    @commands.command(name='join')
    async def join(self, ctx):
        await join_channel(self, ctx)
        print('CONNECTED')


##########################################################################################

def setup(bot):
    bot.add_cog(BotCommands(bot))
