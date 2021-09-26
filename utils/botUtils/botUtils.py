async def join_channel(self, ctx):
    channel = ctx.author.voice.channel
    voice_client_list = self.bot.voice_clients

    if channel and not voice_client_list:
        await channel.connect()
        print('CONNECTED')


async def leave_channel(self, ctx):
    voice_Client = self.bot.voice_clients[0]

    await voice_Client.disconnect()
