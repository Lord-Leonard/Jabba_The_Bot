import traceback

from discord.ext import commands


class MusicBot(commands.Bot):
    inital_extensions = ['cogs.botCommands',
                         'cogs.musicCommands',
                         'cogs.queueCommands',
                         'cogs.testCommand']

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

client.run('ODkwODM1NjQyNTEzMjQ4MjY2.YU1lWA.31tzk_k4_QFCNz8mtDx90eIgk9U')
