import sqlite3

##########################################################################################
from song import Song

con = sqlite3.connect('song.db')
cur = con.cursor()


##########################################################################################


def clear_song_table(ctx):
    sql = 'DELETE FROM songs'

    cur.execute(sql)
    con.commit()

    print('Song Table cleared')


def add_song_to_table(so):
    sql = 'INSERT INTO songs (`URL`, `Title`, `Duration`, `FileName`, `FileLocation`) VALUES (?, ?, ?, ?, ?)'
    song_data = (so.url, so.title, so.duration, so.file_name, so.file_location)

    cur.execute(sql, song_data)
    con.commit()

    print('Song added to Table')


def update_song_location(so):
    sql = 'UPDATE songs SET FileLocation = ? WHERE URL LIKE ?'
    song_data = (so.file_location, so.url)

    cur.execute(sql, song_data)
    con.commit()

    print('Song location updated')


def print_table():
    for row in cur.execute('SELECT * FROM songs'):
        print(row)


def get_song_list():
    sql = 'SELECT * FROM songs'

    song_list = cur.execute(sql)

    return song_list

# def get_song_as_song_object(ctx, url):
#     sql = 'SELECT * FROM songs WHERE URL LIKE ?'
#     song_data = url
#
#     cur.execute(sql, song_data)
#
#     for song in cur:
#         return Song(
#             song[0],
#             song[1],
#             song[2],
#             song[3],
#             song[4]
#         )
#
#     print('Song object created')

##########################################################################################

# def create_song_table(ctx):
#     cur.execute('''
#         CREATE TABLE songs (
#             URL text,
#             Title text,
#             Duration int,
#             FileName text,
#             FileLocation text
#         )
#         ''')


