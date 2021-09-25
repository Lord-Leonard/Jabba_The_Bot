import os

import discord
import youtube_dl

from song_queue import Queue
from song import Song

########################################################################################################################

ydl_opts = {
    'format': 'bestaudio/best',
    'default_search': 'auto',
    'postprocessors': [{
        'key': 'FFmpegExtractAudio',
        'preferredcodec': 'mp3',
        'preferredquality': '192'
    }]
}


########################################################################################################################

def delete_song(ctx, file_location):
    try:
        os.remove(file_location)
        print('Song deleted')
    except PermissionError:
        print('Song konnte nicht gelöscht werden')
        return


def clear_directory(direct):
    for file in os.listdir(direct):
        os.remove(direct + '/' + file)


def download_file(url):
    with youtube_dl.YoutubeDL(ydl_opts) as ydl:
        ydl._ies = [ydl.get_info_extractor('Youtube')]
        print('Downloading audio now\n')
        ydl.download([url])
        print('Download done')


def move_song(so, directory):

    song_name = so.title
    song_location = so.file_name
    song_destination = directory + '/' + song_name

    move_mp3_file(song_destination)

    so.file_location = song_destination


def move_mp3_file(file_destination):
    for file in os.listdir('./'):
        if file.endswith('.mp3'):
            os.rename(file, file_destination)
            print(f'Renamed File: {file}\n')
