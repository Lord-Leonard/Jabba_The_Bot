from discord.ext import commands


##########################################################################################

class HelpCommand(commands.Cog):
    def __init__(self, bot):
        self.bot = bot
        bot.remove_command('help')

    @commands.command(name='help')
    async def help(self, ctx):
        await ctx.send(
            '```Folgende Befehle setehen zur Zeit zur Verfügung ^^ \n \
\n \
+ Bot Befehle: \n \
\t- join: Der Bot betritt den aktuellen Sprachkanal \n \
\t- leave: Der Bot verlässt den aktuellen Sprachkanal \n \
\n \
+ Musik Befehle: \n \
\t- play (optional: URL): \n \
\t\tspielt einen Pausierten Song weiter ab \n \
\t\toder \n \
\t\tspielt den ersten Song in der Warteschlange \n \
\t\toder  \n \
\t\tspielt den Song der URL ab \n \
\t- pause: Pausiert den aktuellen Song \n \
\t- stop: beendet den aktuellen Song \n \
\n \
+ Warteschlangenbefehle \n \
\t- queue(optional: URL) \n \
\t\tgibt die aktuelle Warteschlange aus \n \
\t\tfügt den Song der URL der Warteschlange hinzu \n \
\t- clear queue: leert die Warteschlange```'
        )


##########################################################################################

def setup(bot):
    bot.add_cog(HelpCommand(bot))
