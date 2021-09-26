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
    print(f'Song added to queue')


def add_song_to_song_list(so):
    song_list.append(so)


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
