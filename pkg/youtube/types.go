package youtube

import "encoding/json"

type SearchResult struct {
	Title       string
	Artist      string
	Album       string
	VideoID     string
	CoverArtUrl string
}

type Run struct {
	Text               string                 `json:"text"`
	NavigationEndpoint *RunNavigationEndpoint `json:"navigationEndpoint,omitempty"`
}

type RunNavigationEndpoint struct {
	ClickTrackingParams string             `json:"clickTrackingParams"`
	WatchEndpoint       *RunWatchEndpoint  `json:"watchEndpoint,omitempty"`
	BrowseEndpoint      *RunBrowseEndpoint `json:"browseEndpoint,omitempty"`
	SearchEndpoint      *RunSearchEndpoint `json:"searchEndpoint,omitempty"`
	SignInEndpoint      *RunSignInEndpoint `json:"signInEndpoint,omitempty"`
}

type RunWatchEndpoint struct {
	VideoID                            string          `json:"videoId"`
	PlaylistID                         string          `json:"playlistId,omitempty"`
	Index                              int             `json:"index,omitempty"`
	Params                             string          `json:"params,omitempty"`
	PlayerParams                       string          `json:"playerParams,omitempty"`
	PlaylistSetVideoID                 string          `json:"playlistSetVideoId,omitempty"`
	LoggingContext                     json.RawMessage `json:"loggingContext,omitempty"`
	WatchEndpointMusicSupportedConfigs json.RawMessage `json:"watchEndpointMusicSupportedConfigs,omitempty"`
}

type RunBrowseEndpoint struct {
	BrowseID                              string                                   `json:"browseId"`
	BrowseEndpointContextSupportedConfigs RunBrowseEndpointContextSupportedConfigs `json:"browseEndpointContextSupportedConfigs"`
}

type RunBrowseEndpointContextSupportedConfigs struct {
	BrowseEndpointContextMusicConfig RunBrowseEndpointContextMusicConfig `json:"browseEndpointContextMusicConfig"`
}

type RunBrowseEndpointContextMusicConfig struct {
	PageType string `json:"pageType"`
}

type RunSearchEndpoint struct {
	Query  string `json:"query"`
	Params string `json:"params"`
}

type RunSignInEndpoint struct {
	Hack bool `json:"hack"`
}

type YoutubeSearchPayload struct {
	Context struct {
		Client struct {
			ClientName    string `json:"clientName"`
			ClientVersion string `json:"clientVersion"`
			Hl            string `json:"hl,omitempty"`
		} `json:"client"`
	} `json:"context"`
	VideoID                       string `json:"videoId,omitempty"`
	PlaylistID                    string `json:"playlistId,omitempty"`
	EnablePersistentPlaylistPanel bool   `json:"enablePersistentPlaylistPanel,omitempty"`
	IsAudioOnly                   bool   `json:"isAudioOnly,omitempty"`
	Query                         string `json:"query,omitempty"`
	Params                        string `json:"params,omitempty"`
}

type YoutubeNextResponse struct {
	ResponseContext json.RawMessage `json:"responseContext"`
	Contents        struct {
		SingleColumnMusicWatchNextResultsRenderer struct {
			TabbedRenderer struct {
				WatchNextTabbedResultsRenderer struct {
					Tabs []struct {
						TabRenderer struct {
							Title   string `json:"title"`
							Content struct {
								MusicQueueRenderer struct {
									Content struct {
										PlaylistPanelRenderer struct {
											Contents []struct {
												PlaylistPanelVideoRenderer struct {
													Title struct {
														Runs []Run `json:"runs"`
													} `json:"title"`
													LongBylineText     json.RawMessage `json:"longBylineText"`
													Thumbnail          json.RawMessage `json:"thumbnail"`
													LengthText         json.RawMessage `json:"lengthText"`
													Selected           bool            `json:"selected"`
													NavigationEndpoint struct {
														ClickTrackingParams string `json:"clickTrackingParams"`
														WatchEndpoint       struct {
															VideoID                            string          `json:"videoId"`
															PlaylistID                         string          `json:"playlistId"`
															Index                              int             `json:"index"`
															Params                             string          `json:"params"`
															PlayerParams                       string          `json:"playerParams"`
															PlaylistSetVideoID                 string          `json:"playlistSetVideoId"`
															LoggingContext                     json.RawMessage `json:"loggingContext"`
															WatchEndpointMusicSupportedConfigs json.RawMessage `json:"watchEndpointMusicSupportedConfigs"`
														} `json:"watchEndpoint"`
													} `json:"navigationEndpoint"`
													VideoID                 string          `json:"videoId"`
													ShortBylineText         json.RawMessage `json:"shortBylineText"`
													TrackingParams          string          `json:"trackingParams"`
													Menu                    json.RawMessage `json:"menu"`
													PlaylistSetVideoID      string          `json:"playlistSetVideoId"`
													CanReorder              bool            `json:"canReorder"`
													QueueNavigationEndpoint json.RawMessage `json:"queueNavigationEndpoint"`
												} `json:"playlistPanelVideoRenderer"`
											} `json:"contents"`
											PlaylistID          string          `json:"playlistId"`
											IsInfinite          bool            `json:"isInfinite"`
											Continuations       json.RawMessage `json:"continuations"`
											TrackingParams      string          `json:"trackingParams"`
											NumItemsToShow      int             `json:"numItemsToShow"`
											ShuffleToggleButton json.RawMessage `json:"shuffleToggleButton"`
										} `json:"playlistPanelRenderer"`
									} `json:"content"`
									Hack               bool            `json:"hack"`
									Header             json.RawMessage `json:"header"`
									SubHeaderChipCloud json.RawMessage `json:"subHeaderChipCloud"`
								} `json:"musicQueueRenderer"`
							} `json:"content"`
							TrackingParams string `json:"trackingParams"`
						} `json:"tabRenderer"`
					} `json:"tabs"`
				} `json:"watchNextTabbedResultsRenderer"`
			} `json:"tabbedRenderer"`
		} `json:"singleColumnMusicWatchNextResultsRenderer"`
	} `json:"contents"`
	CurrentVideoEndpoint json.RawMessage `json:"currentVideoEndpoint"`
	TrackingParams       string          `json:"trackingParams"`
	PlayerOverlays       json.RawMessage `json:"playerOverlays"`
	VideoReporting       json.RawMessage `json:"videoReporting"`
	QueueContextParams   string          `json:"queueContextParams"`
}

type ResponseContext struct {
	VisitorData           string `json:"visitorData"`
	ServiceTrackingParams []struct {
		Service string `json:"service"`
		Params  []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"params"`
	} `json:"serviceTrackingParams"`
	MaxAgeSeconds int `json:"maxAgeSeconds"`
}

type ItemSectionRenderer struct {
	Contents []struct {
		MessageRenderer struct {
			TrackingParams string `json:"trackingParams"`
			Button         struct {
				ButtonRenderer struct {
					Style      string `json:"style"`
					IsDisabled bool   `json:"isDisabled"`
					Text       struct {
						SimpleText string `json:"simpleText"`
					} `json:"text"`
					Icon struct {
						IconType string `json:"iconType"`
					} `json:"icon"`
					NavigationEndpoint struct {
						ClickTrackingParams string `json:"clickTrackingParams"`
						URLEndpoint         struct {
							URL    string `json:"url"`
							Target string `json:"target"`
						} `json:"urlEndpoint"`
					} `json:"navigationEndpoint"`
					TrackingParams string `json:"trackingParams"`
					IconPosition   string `json:"iconPosition"`
				} `json:"buttonRenderer"`
			} `json:"button"`
			Style struct {
				Value string `json:"value"`
			} `json:"style"`
		} `json:"messageRenderer"`
	} `json:"contents"`
	TrackingParams string `json:"trackingParams"`
}

type Thumbnail struct {
	MusicThumbnailRenderer struct {
		Thumbnail struct {
			Thumbnails []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"thumbnails"`
		} `json:"thumbnail"`
		ThumbnailCrop  string `json:"thumbnailCrop"`
		ThumbnailScale string `json:"thumbnailScale"`
		TrackingParams string `json:"trackingParams"`
	} `json:"musicThumbnailRenderer"`
}

type MusicResponsiveListItemRenderer struct {
	TrackingParams string `json:"trackingParams"`
	Thumbnail      struct {
		MusicThumbnailRenderer struct {
			Thumbnail struct {
				Thumbnails []struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"thumbnails"`
			} `json:"thumbnail"`
			ThumbnailCrop  string `json:"thumbnailCrop"`
			ThumbnailScale string `json:"thumbnailScale"`
			TrackingParams string `json:"trackingParams"`
		} `json:"musicThumbnailRenderer"`
	} `json:"thumbnail"`
	Overlay struct {
		MusicItemThumbnailOverlayRenderer struct {
			Background struct {
				VerticalGradient struct {
					GradientLayerColors []string `json:"gradientLayerColors"`
				} `json:"verticalGradient"`
			} `json:"background"`
			Content struct {
				MusicPlayButtonRenderer struct {
					PlayNavigationEndpoint struct {
						ClickTrackingParams string `json:"clickTrackingParams"`
						WatchEndpoint       struct {
							VideoID                            string `json:"videoId"`
							WatchEndpointMusicSupportedConfigs struct {
								WatchEndpointMusicConfig struct {
									MusicVideoType string `json:"musicVideoType"`
								} `json:"watchEndpointMusicConfig"`
							} `json:"watchEndpointMusicSupportedConfigs"`
						} `json:"watchEndpoint"`
					} `json:"playNavigationEndpoint"`
					TrackingParams string `json:"trackingParams"`
					PlayIcon       struct {
						IconType string `json:"iconType"`
					} `json:"playIcon"`
					PauseIcon struct {
						IconType string `json:"iconType"`
					} `json:"pauseIcon"`
					IconColor             int64 `json:"iconColor"`
					BackgroundColor       int   `json:"backgroundColor"`
					ActiveBackgroundColor int   `json:"activeBackgroundColor"`
					LoadingIndicatorColor int   `json:"loadingIndicatorColor"`
					PlayingIcon           struct {
						IconType string `json:"iconType"`
					} `json:"playingIcon"`
					IconLoadingColor      int    `json:"iconLoadingColor"`
					ActiveScaleFactor     int    `json:"activeScaleFactor"`
					ButtonSize            string `json:"buttonSize"`
					RippleTarget          string `json:"rippleTarget"`
					AccessibilityPlayData struct {
						AccessibilityData struct {
							Label string `json:"label"`
						} `json:"accessibilityData"`
					} `json:"accessibilityPlayData"`
					AccessibilityPauseData struct {
						AccessibilityData struct {
							Label string `json:"label"`
						} `json:"accessibilityData"`
					} `json:"accessibilityPauseData"`
				} `json:"musicPlayButtonRenderer"`
			} `json:"content"`
			ContentPosition string `json:"contentPosition"`
			DisplayStyle    string `json:"displayStyle"`
		} `json:"musicItemThumbnailOverlayRenderer"`
	} `json:"overlay"`
	FlexColumns []struct {
		MusicResponsiveListItemFlexColumnRenderer struct {
			Text struct {
				Runs []Run `json:"runs"`
			} `json:"text"`
			DisplayPriority string `json:"displayPriority"`
		} `json:"musicResponsiveListItemFlexColumnRenderer"`
	} `json:"flexColumns"`
	Menu struct {
		MenuRenderer struct {
			Items []struct {
				MenuNavigationItemRenderer struct {
					Text struct {
						Runs []Run `json:"runs"`
					} `json:"text"`
					Icon struct {
						IconType string `json:"iconType"`
					} `json:"icon"`
					NavigationEndpoint struct {
						ClickTrackingParams string `json:"clickTrackingParams"`
						WatchEndpoint       struct {
							VideoID        string `json:"videoId"`
							PlaylistID     string `json:"playlistId"`
							Params         string `json:"params"`
							LoggingContext struct {
								VssLoggingContext struct {
									SerializedContextData string `json:"serializedContextData"`
								} `json:"vssLoggingContext"`
							} `json:"loggingContext"`
							WatchEndpointMusicSupportedConfigs struct {
								WatchEndpointMusicConfig struct {
									MusicVideoType string `json:"musicVideoType"`
								} `json:"watchEndpointMusicConfig"`
							} `json:"watchEndpointMusicSupportedConfigs"`
						} `json:"watchEndpoint"`
					} `json:"navigationEndpoint"`
					TrackingParams string `json:"trackingParams"`
				} `json:"menuNavigationItemRenderer,omitempty"`
				MenuServiceItemRenderer struct {
					Text struct {
						Runs []Run `json:"runs"`
					} `json:"text"`
					Icon struct {
						IconType string `json:"iconType"`
					} `json:"icon"`
					ServiceEndpoint struct {
						ClickTrackingParams string `json:"clickTrackingParams"`
						QueueAddEndpoint    struct {
							QueueTarget struct {
								VideoID      string `json:"videoId"`
								OnEmptyQueue struct {
									ClickTrackingParams string `json:"clickTrackingParams"`
									WatchEndpoint       struct {
										VideoID string `json:"videoId"`
									} `json:"watchEndpoint"`
								} `json:"onEmptyQueue"`
							} `json:"queueTarget"`
							QueueInsertPosition string `json:"queueInsertPosition"`
							Commands            []struct {
								ClickTrackingParams string `json:"clickTrackingParams"`
								AddToToastAction    struct {
									Item struct {
										NotificationTextRenderer struct {
											SuccessResponseText struct {
												Runs []Run `json:"runs"`
											} `json:"successResponseText"`
											TrackingParams string `json:"trackingParams"`
										} `json:"notificationTextRenderer"`
									} `json:"item"`
								} `json:"addToToastAction"`
							} `json:"commands"`
						} `json:"queueAddEndpoint"`
					} `json:"serviceEndpoint"`
					TrackingParams string `json:"trackingParams"`
				} `json:"menuServiceItemRenderer,omitempty"`
				ToggleMenuServiceItemRenderer struct {
					DefaultText struct {
						Runs []Run `json:"runs"`
					} `json:"defaultText"`
					DefaultIcon struct {
						IconType string `json:"iconType"`
					} `json:"defaultIcon"`
					DefaultServiceEndpoint struct {
						ClickTrackingParams string `json:"clickTrackingParams"`
						ModalEndpoint       struct {
							Modal struct {
								ModalWithTitleAndButtonRenderer struct {
									Title struct {
										Runs []Run `json:"runs"`
									} `json:"title"`
									Content struct {
										Runs []Run `json:"runs"`
									} `json:"content"`
									Button struct {
										ButtonRenderer struct {
											Style      string `json:"style"`
											IsDisabled bool   `json:"isDisabled"`
											Text       struct {
												Runs []Run `json:"runs"`
											} `json:"text"`
											NavigationEndpoint struct {
												ClickTrackingParams string `json:"clickTrackingParams"`
												SignInEndpoint      struct {
													Hack bool `json:"hack"`
												} `json:"signInEndpoint"`
											} `json:"navigationEndpoint"`
											TrackingParams string `json:"trackingParams"`
										} `json:"buttonRenderer"`
									} `json:"button"`
								} `json:"modalWithTitleAndButtonRenderer"`
							} `json:"modal"`
						} `json:"modalEndpoint"`
					} `json:"defaultServiceEndpoint"`
					ToggledText struct {
						Runs []Run `json:"runs"`
					} `json:"toggledText"`
					ToggledIcon struct {
						IconType string `json:"iconType"`
					} `json:"toggledIcon"`
					TrackingParams string `json:"trackingParams"`
				} `json:"toggleMenuServiceItemRenderer,omitempty"`
				MenuServiceItemDownloadRenderer struct {
					ServiceEndpoint struct {
						ClickTrackingParams  string `json:"clickTrackingParams"`
						OfflineVideoEndpoint struct {
							VideoID      string `json:"videoId"`
							OnAddCommand struct {
								ClickTrackingParams      string `json:"clickTrackingParams"`
								GetDownloadActionCommand struct {
									VideoID string `json:"videoId"`
									Params  string `json:"params"`
								} `json:"getDownloadActionCommand"`
							} `json:"onAddCommand"`
						} `json:"offlineVideoEndpoint"`
					} `json:"serviceEndpoint"`
					TrackingParams string `json:"trackingParams"`
					BadgeIcon      struct {
						IconType string `json:"iconType"`
					} `json:"badgeIcon"`
				} `json:"menuServiceItemDownloadRenderer,omitempty"`
			} `json:"items"`
			TrackingParams string `json:"trackingParams"`
			Accessibility  struct {
				AccessibilityData struct {
					Label string `json:"label"`
				} `json:"accessibilityData"`
			} `json:"accessibility"`
		} `json:"menuRenderer"`
	} `json:"menu"`
	PlaylistItemData struct {
		VideoID string `json:"videoId"`
	} `json:"playlistItemData"`
	FlexColumnDisplayStyle string `json:"flexColumnDisplayStyle"`
	ItemHeight             string `json:"itemHeight"`
}

type Button struct {
	ButtonRenderer struct {
		Style      string `json:"style"`
		Size       string `json:"size"`
		IsDisabled bool   `json:"isDisabled"`
		Text       struct {
			Runs []Run `json:"runs"`
		} `json:"text"`
		Icon struct {
			IconType string `json:"iconType"`
		} `json:"icon"`
		Accessibility struct {
			Label string `json:"label"`
		} `json:"accessibility"`
		TrackingParams    string `json:"trackingParams"`
		AccessibilityData struct {
			AccessibilityData struct {
				Label string `json:"label"`
			} `json:"accessibilityData"`
		} `json:"accessibilityData"`
		Command struct {
			ClickTrackingParams string `json:"clickTrackingParams"`
			WatchEndpoint       struct {
				VideoID                            string `json:"videoId"`
				Params                             string `json:"params"`
				WatchEndpointMusicSupportedConfigs struct {
					WatchEndpointMusicConfig struct {
						MusicVideoType string `json:"musicVideoType"`
					} `json:"watchEndpointMusicConfig"`
				} `json:"watchEndpointMusicSupportedConfigs"`
			} `json:"watchEndpoint"`
		} `json:"command"`
	} `json:"buttonRenderer"`
}

type Menu struct {
	MenuRenderer struct {
		Items []struct {
			MenuNavigationItemRenderer struct {
				Text struct {
					Runs []Run `json:"runs"`
				} `json:"text"`
				Icon struct {
					IconType string `json:"iconType"`
				} `json:"icon"`
				NavigationEndpoint struct {
					ClickTrackingParams string `json:"clickTrackingParams"`
					WatchEndpoint       struct {
						VideoID        string `json:"videoId"`
						PlaylistID     string `json:"playlistId"`
						Params         string `json:"params"`
						LoggingContext struct {
							VssLoggingContext struct {
								SerializedContextData string `json:"serializedContextData"`
							} `json:"vssLoggingContext"`
						} `json:"loggingContext"`
						WatchEndpointMusicSupportedConfigs struct {
							WatchEndpointMusicConfig struct {
								MusicVideoType string `json:"musicVideoType"`
							} `json:"watchEndpointMusicConfig"`
						} `json:"watchEndpointMusicSupportedConfigs"`
					} `json:"watchEndpoint"`
				} `json:"navigationEndpoint"`
				TrackingParams string `json:"trackingParams"`
			} `json:"menuNavigationItemRenderer,omitempty"`
			MenuServiceItemRenderer struct {
				Text struct {
					Runs []Run `json:"runs"`
				} `json:"text"`
				Icon struct {
					IconType string `json:"iconType"`
				} `json:"icon"`
				ServiceEndpoint struct {
					ClickTrackingParams string `json:"clickTrackingParams"`
					QueueAddEndpoint    struct {
						QueueTarget struct {
							VideoID      string `json:"videoId"`
							OnEmptyQueue struct {
								ClickTrackingParams string `json:"clickTrackingParams"`
								WatchEndpoint       struct {
									VideoID string `json:"videoId"`
								} `json:"watchEndpoint"`
							} `json:"onEmptyQueue"`
						} `json:"queueTarget"`
						QueueInsertPosition string `json:"queueInsertPosition"`
						Commands            []struct {
							ClickTrackingParams string `json:"clickTrackingParams"`
							AddToToastAction    struct {
								Item struct {
									NotificationTextRenderer struct {
										SuccessResponseText struct {
											Runs []Run `json:"runs"`
										} `json:"successResponseText"`
										TrackingParams string `json:"trackingParams"`
									} `json:"notificationTextRenderer"`
								} `json:"item"`
							} `json:"addToToastAction"`
						} `json:"commands"`
					} `json:"queueAddEndpoint"`
				} `json:"serviceEndpoint"`
				TrackingParams string `json:"trackingParams"`
			} `json:"menuServiceItemRenderer,omitempty"`
			ToggleMenuServiceItemRenderer struct {
				DefaultText struct {
					Runs []Run `json:"runs"`
				} `json:"defaultText"`
				DefaultIcon struct {
					IconType string `json:"iconType"`
				} `json:"defaultIcon"`
				DefaultServiceEndpoint struct {
					ClickTrackingParams string `json:"clickTrackingParams"`
					ModalEndpoint       struct {
						Modal struct {
							ModalWithTitleAndButtonRenderer struct {
								Title struct {
									Runs []Run `json:"runs"`
								} `json:"title"`
								Content struct {
									Runs []Run `json:"runs"`
								} `json:"content"`
								Button struct {
									ButtonRenderer struct {
										Style      string `json:"style"`
										IsDisabled bool   `json:"isDisabled"`
										Text       struct {
											Runs []Run `json:"runs"`
										} `json:"text"`
										NavigationEndpoint struct {
											ClickTrackingParams string `json:"clickTrackingParams"`
											SignInEndpoint      struct {
												Hack bool `json:"hack"`
											} `json:"signInEndpoint"`
										} `json:"navigationEndpoint"`
										TrackingParams string `json:"trackingParams"`
									} `json:"buttonRenderer"`
								} `json:"button"`
							} `json:"modalWithTitleAndButtonRenderer"`
						} `json:"modal"`
					} `json:"modalEndpoint"`
				} `json:"defaultServiceEndpoint"`
				ToggledText struct {
					Runs []Run `json:"runs"`
				} `json:"toggledText"`
				ToggledIcon struct {
					IconType string `json:"iconType"`
				} `json:"toggledIcon"`
				TrackingParams string `json:"trackingParams"`
			} `json:"toggleMenuServiceItemRenderer,omitempty"`
			MenuServiceItemDownloadRenderer struct {
				ServiceEndpoint struct {
					ClickTrackingParams  string `json:"clickTrackingParams"`
					OfflineVideoEndpoint struct {
						VideoID      string `json:"videoId"`
						OnAddCommand struct {
							ClickTrackingParams      string `json:"clickTrackingParams"`
							GetDownloadActionCommand struct {
								VideoID string `json:"videoId"`
								Params  string `json:"params"`
							} `json:"getDownloadActionCommand"`
						} `json:"onAddCommand"`
					} `json:"offlineVideoEndpoint"`
				} `json:"serviceEndpoint"`
				TrackingParams string `json:"trackingParams"`
				BadgeIcon      struct {
					IconType string `json:"iconType"`
				} `json:"badgeIcon"`
			} `json:"menuServiceItemDownloadRenderer,omitempty"`
		} `json:"items"`
		TrackingParams string `json:"trackingParams"`
		Accessibility  struct {
			AccessibilityData struct {
				Label string `json:"label"`
			} `json:"accessibilityData"`
		} `json:"accessibility"`
	} `json:"menuRenderer"`
}

type ThumbnailOverlay struct {
	MusicItemThumbnailOverlayRenderer struct {
		Background struct {
			VerticalGradient struct {
				GradientLayerColors []string `json:"gradientLayerColors"`
			} `json:"verticalGradient"`
		} `json:"background"`
		Content struct {
			MusicPlayButtonRenderer struct {
				PlayNavigationEndpoint struct {
					ClickTrackingParams string `json:"clickTrackingParams"`
					WatchEndpoint       struct {
						VideoID                            string `json:"videoId"`
						WatchEndpointMusicSupportedConfigs struct {
							WatchEndpointMusicConfig struct {
								MusicVideoType string `json:"musicVideoType"`
							} `json:"watchEndpointMusicConfig"`
						} `json:"watchEndpointMusicSupportedConfigs"`
					} `json:"watchEndpoint"`
				} `json:"playNavigationEndpoint"`
				TrackingParams string `json:"trackingParams"`
				PlayIcon       struct {
					IconType string `json:"iconType"`
				} `json:"playIcon"`
				PauseIcon struct {
					IconType string `json:"iconType"`
				} `json:"pauseIcon"`
				IconColor             int64 `json:"iconColor"`
				BackgroundColor       int   `json:"backgroundColor"`
				ActiveBackgroundColor int   `json:"activeBackgroundColor"`
				LoadingIndicatorColor int   `json:"loadingIndicatorColor"`
				PlayingIcon           struct {
					IconType string `json:"iconType"`
				} `json:"playingIcon"`
				IconLoadingColor      int    `json:"iconLoadingColor"`
				ActiveScaleFactor     int    `json:"activeScaleFactor"`
				ButtonSize            string `json:"buttonSize"`
				RippleTarget          string `json:"rippleTarget"`
				AccessibilityPlayData struct {
					AccessibilityData struct {
						Label string `json:"label"`
					} `json:"accessibilityData"`
				} `json:"accessibilityPlayData"`
				AccessibilityPauseData struct {
					AccessibilityData struct {
						Label string `json:"label"`
					} `json:"accessibilityData"`
				} `json:"accessibilityPauseData"`
			} `json:"musicPlayButtonRenderer"`
		} `json:"content"`
		ContentPosition string `json:"contentPosition"`
		DisplayStyle    string `json:"displayStyle"`
	} `json:"musicItemThumbnailOverlayRenderer"`
}

type SearchResponse struct {
	ResponseContext json.RawMessage `json:"responseContext"`
	Contents        struct {
		TabbedSearchResultsRenderer struct {
			Tabs []struct {
				TabRenderer struct {
					Title    string `json:"title"`
					Selected bool   `json:"selected"`
					Content  struct {
						SectionListRenderer struct {
							Contents []struct {
								ItemSectionRenderer    json.RawMessage `json:"itemSectionRenderer,omitempty"`
								MusicCardShelfRenderer *struct {
									TrackingParams string    `json:"trackingParams"`
									Thumbnail      Thumbnail `json:"thumbnail"`
									Title          struct {
										Runs []Run `json:"runs"`
									} `json:"title"`
									Subtitle `json:"subtitle"`
									Contents []struct {
										MessageRenderer struct {
											Text struct {
												Runs []Run `json:"runs"`
											} `json:"text"`
											TrackingParams string `json:"trackingParams"`
											Style          struct {
												Value string `json:"value"`
											} `json:"style"`
										} `json:"messageRenderer,omitempty"`
										MusicResponsiveListItemRenderer `json:"musicResponsiveListItemRenderer,omitempty"`
									} `json:"contents"`
									Buttons json.RawMessage `json:"buttons"`
									Menu    json.RawMessage `json:"menu"`
									OnTap   struct {
										ClickTrackingParams string `json:"clickTrackingParams"`
										WatchEndpoint       struct {
											VideoID                            string `json:"videoId"`
											WatchEndpointMusicSupportedConfigs struct {
												WatchEndpointMusicConfig struct {
													MusicVideoType string `json:"musicVideoType"`
												} `json:"watchEndpointMusicConfig"`
											} `json:"watchEndpointMusicSupportedConfigs"`
										} `json:"watchEndpoint"`
									} `json:"onTap"`
									ThumbnailOverlay json.RawMessage `json:"thumbnailOverlay"`
								} `json:"musicCardShelfRenderer,omitempty"`
								MusicShelfRenderer *struct {
									Contents []struct {
										MusicResponsiveListItemRenderer struct {
											TrackingParams string `json:"trackingParams"`
											Thumbnail      struct {
												MusicThumbnailRenderer struct {
													Thumbnail struct {
														Thumbnails []struct {
															URL    string `json:"url"`
															Width  int    `json:"width"`
															Height int    `json:"height"`
														} `json:"thumbnails"`
													} `json:"thumbnail"`
													ThumbnailCrop  string `json:"thumbnailCrop"`
													ThumbnailScale string `json:"thumbnailScale"`
													TrackingParams string `json:"trackingParams"`
												} `json:"musicThumbnailRenderer"`
											} `json:"thumbnail"`
											Overlay struct {
												MusicItemThumbnailOverlayRenderer struct {
													Background struct {
														VerticalGradient struct {
															GradientLayerColors []string `json:"gradientLayerColors"`
														} `json:"verticalGradient"`
													} `json:"background"`
													Content struct {
														MusicPlayButtonRenderer json.RawMessage `json:"musicPlayButtonRenderer"`
													} `json:"content"`
													ContentPosition string `json:"contentPosition"`
													DisplayStyle    string `json:"displayStyle"`
												} `json:"musicItemThumbnailOverlayRenderer"`
											} `json:"overlay"`
											FlexColumns []struct {
												MusicResponsiveListItemFlexColumnRenderer struct {
													Text struct {
														Runs []Run `json:"runs"`
													} `json:"text"`
													DisplayPriority string `json:"displayPriority"`
												} `json:"musicResponsiveListItemFlexColumnRenderer"`
											} `json:"flexColumns"`
											Menu             json.RawMessage `json:"menu"`
											PlaylistItemData struct {
												VideoID string `json:"videoId"`
											} `json:"playlistItemData"`
											FlexColumnDisplayStyle string `json:"flexColumnDisplayStyle"`
											ItemHeight             string `json:"itemHeight"`
										} `json:"musicResponsiveListItemRenderer"`
									} `json:"contents"`
									TrackingParams string `json:"trackingParams"`
									ShelfDivider   struct {
										MusicShelfDividerRenderer struct {
											Hidden bool `json:"hidden"`
										} `json:"musicShelfDividerRenderer"`
									} `json:"shelfDivider"`
								} `json:"musicShelfRenderer,omitempty"`
							} `json:"contents"`
							TrackingParams string          `json:"trackingParams"`
							Header         json.RawMessage `json:"header"`
						} `json:"sectionListRenderer"`
					} `json:"content"`
					TabIdentifier  string `json:"tabIdentifier"`
					TrackingParams string `json:"trackingParams"`
				} `json:"tabRenderer"`
			} `json:"tabs"`
		} `json:"tabbedSearchResultsRenderer"`
	} `json:"contents"`
	TrackingParams string `json:"trackingParams"`
}

type Subtitle struct {
	Runs          []Run `json:"runs"`
	Accessibility struct {
		AccessibilityData struct {
			Label string `json:"label"`
		} `json:"accessibilityData"`
	} `json:"accessibility"`
}

type MusicPlayButtonRenderer struct {
	PlayNavigationEndpoint struct {
		ClickTrackingParams string `json:"clickTrackingParams"`
		WatchEndpoint       struct {
			VideoID                            string `json:"videoId"`
			WatchEndpointMusicSupportedConfigs struct {
				WatchEndpointMusicConfig struct {
					MusicVideoType string `json:"musicVideoType"`
				} `json:"watchEndpointMusicConfig"`
			} `json:"watchEndpointMusicSupportedConfigs"`
		} `json:"watchEndpoint"`
	} `json:"playNavigationEndpoint"`
	TrackingParams string `json:"trackingParams"`
	PlayIcon       struct {
		IconType string `json:"iconType"`
	} `json:"playIcon"`
	PauseIcon struct {
		IconType string `json:"iconType"`
	} `json:"pauseIcon"`
	IconColor             int64 `json:"iconColor"`
	BackgroundColor       int   `json:"backgroundColor"`
	ActiveBackgroundColor int   `json:"activeBackgroundColor"`
	LoadingIndicatorColor int   `json:"loadingIndicatorColor"`
	PlayingIcon           struct {
		IconType string `json:"iconType"`
	} `json:"playingIcon"`
	IconLoadingColor      int    `json:"iconLoadingColor"`
	ActiveScaleFactor     int    `json:"activeScaleFactor"`
	ButtonSize            string `json:"buttonSize"`
	RippleTarget          string `json:"rippleTarget"`
	AccessibilityPlayData struct {
		AccessibilityData struct {
			Label string `json:"label"`
		} `json:"accessibilityData"`
	} `json:"accessibilityPlayData"`
	AccessibilityPauseData struct {
		AccessibilityData struct {
			Label string `json:"label"`
		} `json:"accessibilityData"`
	} `json:"accessibilityPauseData"`
}

type Header struct {
	ChipCloudRenderer struct {
		Chips []struct {
			ChipCloudChipRenderer struct {
				Style struct {
					StyleType string `json:"styleType"`
				} `json:"style"`
				Text struct {
					Runs []Run `json:"runs"`
				} `json:"text"`
				NavigationEndpoint struct {
					ClickTrackingParams string `json:"clickTrackingParams"`
					SearchEndpoint      struct {
						Query  string `json:"query"`
						Params string `json:"params"`
					} `json:"searchEndpoint"`
				} `json:"navigationEndpoint"`
				TrackingParams    string `json:"trackingParams"`
				AccessibilityData struct {
					AccessibilityData struct {
						Label string `json:"label"`
					} `json:"accessibilityData"`
				} `json:"accessibilityData"`
				IsSelected bool   `json:"isSelected"`
				UniqueID   string `json:"uniqueId"`
			} `json:"chipCloudChipRenderer"`
		} `json:"chips"`
		CollapsedRowCount    int    `json:"collapsedRowCount"`
		TrackingParams       string `json:"trackingParams"`
		HorizontalScrollable bool   `json:"horizontalScrollable"`
	} `json:"chipCloudRenderer"`
}
