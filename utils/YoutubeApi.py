# -*- coding: utf-8 -*-

# Sample Python code for youtube.channels.list
# See instructions for running these code samples locally:
# https://developers.google.com/explorer-help/guides/code_samples#python

import os

import googleapiclient.discovery


def getSourceVideoID(source):
    url = source.web_url
    videoId = url.removeprefix('https://www.youtube.com/watch?v=')
    return videoId


def getRelatedVideoData(currentvideoid, amount):
    # Disable OAuthlib's HTTPS verification when running locally.
    # *DO NOT* leave this option enabled in production.
    os.environ["OAUTHLIB_INSECURE_TRANSPORT"] = "1"

    api_service_name = "youtube"
    api_version = "v3"
    DEVELOPER_KEY = "AIzaSyBmcyHnoniLJx96ThX5FqZmDurcdOnyYcc"
    # DEVELOPER_KEY = "AIzaSyBuCrm94Tc2AS8FXbAQzqdmEw2XQ50EAHc"

    youtube = googleapiclient.discovery.build(
        api_service_name, api_version, developerKey=DEVELOPER_KEY)

    request = youtube.search().list(
        part="snippet",
        maxResults=amount,
        order="relevance",
        relatedToVideoId=currentvideoid,
        safeSearch="none",
        type="video",
        videoCategoryId="10"
    )
    response = request.execute()

    return response


def getRelatedVideoIdList(relatedvideodatalist):
    relatedVideoIdList = []

    for item in relatedvideodatalist['items']:
        relatedVideoIdList.append(item['id']['videoId'])
    return relatedVideoIdList


def createRelatedVideoUrlList(relatedvideoidlist):
    youtubeBaseUrl = 'https://www.youtube.com/watch?v='
    relatedVideoUrlList = []

    for url in relatedvideoidlist:
        relatedVideoUrl = youtubeBaseUrl + url
        relatedVideoUrlList.append(relatedVideoUrl)

    return relatedVideoUrlList


def getRelatedVideoUrlList(source, amount):
    currentVideoId = getSourceVideoID(source)
    relatedVideoDataList = getRelatedVideoData(currentVideoId, amount)
    relatedVideoIdList = getRelatedVideoIdList(relatedVideoDataList)
    relatedVideoUrlList = createRelatedVideoUrlList(relatedVideoIdList)
    return relatedVideoUrlList
