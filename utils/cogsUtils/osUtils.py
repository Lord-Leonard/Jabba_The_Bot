import os

import youtube_dl

##########################################################################################


ydl_opts = {
    'format': 'bestaudio/best',
    'postprocessors': [{
        'key': 'FFmpegExtractAudio',
        'preferredcodec': 'mp3',
    }]
}

directory_songs = 'songs/'


##########################################################################################

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
        print('Downloading audio now')
        ydl.download([url])
        print('Download done')


def move_download_to_songs(so):
    file_destination = directory_songs + so.title + '.mp3'

    for file in os.listdir('./'):
        if file.endswith('.mp3'):
            os.rename(file, file_destination)
            print(f'Renamed File: {file}\n')

    so.file_location = file_destination
