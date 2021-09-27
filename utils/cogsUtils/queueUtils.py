import json

from utils.cogsUtils.osUtils import move_download_to_songs
from utils.song_queue import song_queue, song_list

##########################################################################################

directory_played = 'played/'
directory_playing = 'playing/'
directory_queued = 'queued/'


##########################################################################################

def add_song_to_queue(so):
    song_queue.append(so)
    move_download_to_songs(so)


def get_song_json_string(so):
    song_list_string = {'song': []}
    song_list_string['song'].append({
        'url': so.url,
        'title': so.title,
        'duration': so.duration,
        'file name': so.file_name,
        'file location': so.file_location
    })
    return song_list_string


def add_song_to_runtime_song_list(so):
    song_list.append(so)
    print('song added to runtime songlist')


def add_song_to_project_song_list(so):
    song_data = get_song_json_string(so)

    with open('song_list.json', 'w', encoding='utf-8') as f:

        data = json.load(f)
        data.append(song_data)
        f.seek(0)
        json.dump(song_data, f, indent=5)
    print('song added to project songlist')


def song_is_on_list(url):
    for song in song_list:
        if song.url == url:
            return True
    return False


def get_song_object_from_list(url):
    for song in song_list:
        if song.url == url:
            return song
        print('i swear to god the song was there 2 seconds ago')


def remove_song_from_queue():
    song_queue.pop(0)

    print(f'Song removed from queue')


def get_top_of_queue():
    return song_queue[0]


def queue_is_empty():
    return not song_queue


def clear_queue():
    song_queue.clear()
