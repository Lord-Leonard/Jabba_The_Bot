async def join_channel(self, ctx):
    channel = ctx.author.voice.channel
    voice_client_list = self.bot.voice_clients

    if channel and not voice_client_list:
        await channel.connect()
        print('CONNECTED')