import os

import discord
import youtube_dl

from utils.cogsUtils.osUtils import *
from utils.cogsUtils.queueUtils import *
from song_queue import Queue
from song import Song


########################################################################################################################


########################################################################################################################


def get_song_info(ctx, url):
    with youtube_dl.YoutubeDL(ydl_opts) as ydl:
        return ydl.extract_info(url, download=False)


def create_song_object(ctx, url):
    song_info = get_song_info(ctx, url)
    return Song(url, song_info['title'], song_info['duration'])


def play_song(vc, ctx):
    vc.play(discord.FFmpegPCMAudio(Queue.song_queue[0].file_location),
            after=lambda e: delete_song_and_check_for_next(ctx, Queue.song_queue[0].file_location, vc))

    print('Music is playing now')


def delete_song_and_check_for_next(ctx, file_location, vc):
    delete_song(ctx, file_location)
    remove_song_from_queue()

    if Queue.song_queue:
        play_song(vc, ctx)
