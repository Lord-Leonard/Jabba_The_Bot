class Song:
    def __init__(self, url, t, dur):
        self.url = url
        self.title = t
        self.duration = dur
        self.file_name = f'{self.title}.mp3'
        self.file_location = self.file_name

    def __str__(self):
        return 'Title: ' + self.title + \
               ', Duration: ' + str(self.duration) + \
               ', Dateiname: ' + self.file_name + \
               ', Url: ' + self.url
