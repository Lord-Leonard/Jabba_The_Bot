
class Song:
    def __init__(self, url, t, dur, file_name='', file_location=''):
        self.url = url
        self.title = t
        self.duration = dur

        if file_name:
            self.file_name = file_name
        else:
            self.file_name = f'{self.title}.mp3'

        if file_location:
            self.file_location = file_location
        else:
            self.file_location = self.file_name

    def __str__(self):
        return 'Title: ' + self.title + \
               ', Duration: ' + str(self.duration) + \
               ', Dateiname: ' + self.file_name + \
               ', Url: ' + self.url
