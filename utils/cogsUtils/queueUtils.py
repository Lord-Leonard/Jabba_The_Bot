from utils.cogsUtils.musicUtils import *
from utils.cogsUtils.osUtils import *
import os

import discord
import youtube_dl

from song_queue import Queue
from song import Song


########################################################################################################################


########################################################################################################################

def add_song_to_queue(so):
    Queue.song_queue.append(so)
    print(f'Song added to queue')


def remove_song_from_queue():
    Queue.song_queue.pop(0)

    print(f'Song removed from queue')
