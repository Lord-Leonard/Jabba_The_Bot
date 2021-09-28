import csv
import json

import discord
import youtube_dl

from song import Song
from utils.cogsUtils.osUtils import download_file, ydl_opts
from utils.cogsUtils.queueUtils import add_song_to_queue, add_song_to_runtime_song_list, song_queue, \
    remove_song_from_queue, \
    song_is_on_list, get_song_object_from_list, add_song_to_project_song_list

##########################################################################################
from utils.db_Utils import get_song_list

directory_played = 'played/'
directory_playing = 'playing/'


##########################################################################################

def load_song_list():
    count = 0
    song_list = get_song_list()

    for song in song_list:
        song_object = Song(
            song[0],
            song[1],
            song[2],
            song[3],
            song[4]
        )
        add_song_to_runtime_song_list(song_object)
        count += 1

    print(f'Es wurden {count} Songs geladen')


def get_song(ctx, url):
    if song_is_on_list(url):
        song_object = get_song_object_from_list(url)
        add_song_to_queue(song_object)
    else:
        song_object = create_song_object(url)
        print('songobject created')
        download_file(url)
        print('file downloaded')

        add_song_to_queue(song_object)

        add_song_to_runtime_song_list(song_object)
        add_song_to_project_song_list(song_object)


def get_song_info(url):
    with youtube_dl.YoutubeDL(ydl_opts) as ydl:
        return ydl.extract_info(url, download=False)


def create_song_object(url, title='', duration='', file_name='', file_location=''):
    if title and duration and file_name and file_location:
        return Song(url, title, duration, file_name, file_location)
    else:
        song_info = get_song_info(url)
        return Song(url, song_info['title'], song_info['duration'])


def play_song(ctx, vc):
    song_object = song_queue[0]
    vc.play(discord.FFmpegPCMAudio(song_object.file_location),
            after=lambda e: after_song(ctx, vc))

    print('Music is playing now')


def after_song(ctx, vc):
    remove_song_from_queue()

    if song_queue:
        play_song(ctx, vc)


def music_is_playing(voice_client):
    return voice_client.is_playing()


def music_is_paused(voice_client):
    return voice_client.is_paused()


def resume_song(voice_client):
    voice_client.resume()
