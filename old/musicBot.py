import configparser
import traceback

from discord.ext import commands

##########################################################################################

config = configparser.ConfigParser()
config.read('config.ini')
discord_token = config['discord']['Token']


##########################################################################################

class MusicBot(commands.Bot):
    inital_extensions = ['cogs.music']

    async def on_ready(self):
        print(f'Logged in as {self.user}')

    async def on_message(self, message):
        print(f'Message from {message.author}: {message.content}')
        await self.process_commands(message)


client = MusicBot(command_prefix=commands.when_mentioned_or('!'))

if __name__ == '__main__':
    for extesions in client.inital_extensions:
        try:
            client.load_extension(extesions)
        except Exception as e:
            traceback.print_exc()

client.run(discord_token)

