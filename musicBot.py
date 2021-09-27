import configparser
import traceback

from discord.ext import commands
from utils.cogsUtils.musicUtils import load_song_list

##########################################################################################
import utils.song_queue

config = configparser.ConfigParser()
config.read('config.ini')
discord_token = config['discord']['Token']


##########################################################################################

class MusicBot(commands.Bot):
    inital_extensions = ['cogs.botCommands',
                         'cogs.musicCommands',
                         'cogs.queueCommands',
                         'cogs.testCommand',
                         'cogs.helpCommand']

    async def on_ready(self):
        print(f'Logged in as {self.user}')
        load_song_list()

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
