package ripper

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/abema/go-mp4"
	"github.com/grafov/m3u8"
)

func (s *SongInfo) Duration() (ret uint64) {
	for i := range s.samples {
		ret += uint64(s.samples[i].duration)
	}
	return
}

func (*Alac) GetType() mp4.BoxType {
	return BoxTypeAlac()
}

func fileExists(path string) (bool, error) {
	f, err := os.Stat(path)
	if err == nil {
		return !f.IsDir(), nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func writeM4a(w *mp4.Writer, info *SongInfo, meta *AutoGenerated, data []byte, trackNum, trackTotal int) error {
	index := trackNum - 1
	{ // ftyp
		box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeFtyp()})
		if err != nil {
			return err
		}
		_, err = mp4.Marshal(w, &mp4.Ftyp{
			MajorBrand:   [4]byte{'M', '4', 'A', ' '},
			MinorVersion: 0,
			CompatibleBrands: []mp4.CompatibleBrandElem{
				{CompatibleBrand: [4]byte{'M', '4', 'A', ' '}},
				{CompatibleBrand: [4]byte{'m', 'p', '4', '2'}},
				{CompatibleBrand: mp4.BrandISOM()},
				{CompatibleBrand: [4]byte{0, 0, 0, 0}},
			},
		}, box.Context)
		if err != nil {
			return err
		}
		_, err = w.EndBox()
		if err != nil {
			return err
		}
	}

	const chunkSize uint32 = 5
	duration := info.Duration()
	numSamples := uint32(len(info.samples))
	var stco *mp4.BoxInfo

	{ // moov
		_, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMoov()})
		if err != nil {
			return err
		}
		box, err := mp4.ExtractBox(info.r, nil, mp4.BoxPath{mp4.BoxTypeMoov()})
		if err != nil {
			return err
		}
		moovOri := box[0]

		{ // mvhd
			_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMvhd()})
			if err != nil {
				return err
			}

			oriBox, err := mp4.ExtractBoxWithPayload(info.r, moovOri, mp4.BoxPath{mp4.BoxTypeMvhd()})
			if err != nil {
				return err
			}
			mvhd := oriBox[0].Payload.(*mp4.Mvhd)
			if mvhd.Version == 0 {
				mvhd.DurationV0 = uint32(duration)
			} else {
				mvhd.DurationV1 = duration
			}

			_, err = mp4.Marshal(w, mvhd, oriBox[0].Info.Context)
			if err != nil {
				return err
			}

			_, err = w.EndBox()
			if err != nil {
				return err
			}
		}

		{ // trak
			_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeTrak()})
			if err != nil {
				return err
			}

			box, err := mp4.ExtractBox(info.r, moovOri, mp4.BoxPath{mp4.BoxTypeTrak()})
			if err != nil {
				return err
			}
			trakOri := box[0]

			{ // tkhd
				_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeTkhd()})
				if err != nil {
					return err
				}

				oriBox, err := mp4.ExtractBoxWithPayload(info.r, trakOri, mp4.BoxPath{mp4.BoxTypeTkhd()})
				if err != nil {
					return err
				}
				tkhd := oriBox[0].Payload.(*mp4.Tkhd)
				if tkhd.Version == 0 {
					tkhd.DurationV0 = uint32(duration)
				} else {
					tkhd.DurationV1 = duration
				}
				tkhd.SetFlags(0x7)

				_, err = mp4.Marshal(w, tkhd, oriBox[0].Info.Context)
				if err != nil {
					return err
				}

				_, err = w.EndBox()
				if err != nil {
					return err
				}
			}

			{ // mdia
				_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMdia()})
				if err != nil {
					return err
				}

				box, err := mp4.ExtractBox(info.r, trakOri, mp4.BoxPath{mp4.BoxTypeMdia()})
				if err != nil {
					return err
				}
				mdiaOri := box[0]

				{ // mdhd
					_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMdhd()})
					if err != nil {
						return err
					}

					oriBox, err := mp4.ExtractBoxWithPayload(info.r, mdiaOri, mp4.BoxPath{mp4.BoxTypeMdhd()})
					if err != nil {
						return err
					}
					mdhd := oriBox[0].Payload.(*mp4.Mdhd)
					if mdhd.Version == 0 {
						mdhd.DurationV0 = uint32(duration)
					} else {
						mdhd.DurationV1 = duration
					}

					_, err = mp4.Marshal(w, mdhd, oriBox[0].Info.Context)
					if err != nil {
						return err
					}

					_, err = w.EndBox()
					if err != nil {
						return err
					}
				}

				{ // hdlr
					oriBox, err := mp4.ExtractBox(info.r, mdiaOri, mp4.BoxPath{mp4.BoxTypeHdlr()})
					if err != nil {
						return err
					}

					err = w.CopyBox(info.r, oriBox[0])
					if err != nil {
						return err
					}
				}

				{ // minf
					_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMinf()})
					if err != nil {
						return err
					}

					box, err := mp4.ExtractBox(info.r, mdiaOri, mp4.BoxPath{mp4.BoxTypeMinf()})
					if err != nil {
						return err
					}
					minfOri := box[0]

					{ // smhd, dinf
						boxes, err := mp4.ExtractBoxes(info.r, minfOri, []mp4.BoxPath{
							{mp4.BoxTypeSmhd()},
							{mp4.BoxTypeDinf()},
						})
						if err != nil {
							return err
						}

						for _, b := range boxes {
							err = w.CopyBox(info.r, b)
							if err != nil {
								return err
							}
						}
					}

					{ // stbl
						_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStbl()})
						if err != nil {
							return err
						}

						{ // stsd
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStsd()})
							if err != nil {
								return err
							}
							_, err = mp4.Marshal(w, &mp4.Stsd{EntryCount: 1}, box.Context)
							if err != nil {
								return err
							}

							{ // alac
								_, err = w.StartBox(&mp4.BoxInfo{Type: BoxTypeAlac()})
								if err != nil {
									return err
								}

								_, err = w.Write([]byte{
									0, 0, 0, 0, 0, 0, 0, 1,
									0, 0, 0, 0, 0, 0, 0, 0})
								if err != nil {
									return err
								}

								err = binary.Write(w, binary.BigEndian, uint16(info.alacParam.NumChannels))
								if err != nil {
									return err
								}

								err = binary.Write(w, binary.BigEndian, uint16(info.alacParam.BitDepth))
								if err != nil {
									return err
								}

								_, err = w.Write([]byte{0, 0})
								if err != nil {
									return err
								}

								err = binary.Write(w, binary.BigEndian, info.alacParam.SampleRate)
								if err != nil {
									return err
								}

								_, err = w.Write([]byte{0, 0})
								if err != nil {
									return err
								}

								box, err := w.StartBox(&mp4.BoxInfo{Type: BoxTypeAlac()})
								if err != nil {
									return err
								}

								_, err = mp4.Marshal(w, info.alacParam, box.Context)
								if err != nil {
									return err
								}

								_, err = w.EndBox()
								if err != nil {
									return err
								}

								_, err = w.EndBox()
								if err != nil {
									return err
								}
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{ // stts
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStts()})
							if err != nil {
								return err
							}

							var stts mp4.Stts
							for _, sample := range info.samples {
								if len(stts.Entries) != 0 {
									last := &stts.Entries[len(stts.Entries)-1]
									if last.SampleDelta == sample.duration {
										last.SampleCount++
										continue
									}
								}
								stts.Entries = append(stts.Entries, mp4.SttsEntry{
									SampleCount: 1,
									SampleDelta: sample.duration,
								})
							}
							stts.EntryCount = uint32(len(stts.Entries))

							_, err = mp4.Marshal(w, &stts, box.Context)
							if err != nil {
								return err
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{ // stsc
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStsc()})
							if err != nil {
								return err
							}

							if numSamples%chunkSize == 0 {
								_, err = mp4.Marshal(w, &mp4.Stsc{
									EntryCount: 1,
									Entries: []mp4.StscEntry{
										{
											FirstChunk:             1,
											SamplesPerChunk:        chunkSize,
											SampleDescriptionIndex: 1,
										},
									},
								}, box.Context)

								if err != nil {
									return err
								}
							} else {
								_, err = mp4.Marshal(w, &mp4.Stsc{
									EntryCount: 2,
									Entries: []mp4.StscEntry{
										{
											FirstChunk:             1,
											SamplesPerChunk:        chunkSize,
											SampleDescriptionIndex: 1,
										}, {
											FirstChunk:             numSamples/chunkSize + 1,
											SamplesPerChunk:        numSamples % chunkSize,
											SampleDescriptionIndex: 1,
										},
									},
								}, box.Context)

								if err != nil {
									return err
								}
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{ // stsz
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStsz()})
							if err != nil {
								return err
							}

							stsz := mp4.Stsz{SampleCount: numSamples}
							for _, sample := range info.samples {
								stsz.EntrySize = append(stsz.EntrySize, uint32(len(sample.data)))
							}

							_, err = mp4.Marshal(w, &stsz, box.Context)
							if err != nil {
								return err
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{ // stco
							box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeStco()})
							if err != nil {
								return err
							}

							l := (numSamples + chunkSize - 1) / chunkSize
							_, err = mp4.Marshal(w, &mp4.Stco{
								EntryCount:  l,
								ChunkOffset: make([]uint32, l),
							}, box.Context)

							if err != nil {
								return err
							}

							stco, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						_, err = w.EndBox()
						if err != nil {
							return err
						}
					}

					_, err = w.EndBox()
					if err != nil {
						return err
					}
				}

				_, err = w.EndBox()
				if err != nil {
					return err
				}
			}

			_, err = w.EndBox()
			if err != nil {
				return err
			}
		}

		{ // udta
			ctx := mp4.Context{UnderUdta: true}
			_, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeUdta(), Context: ctx})
			if err != nil {
				return err
			}

			{ // meta
				ctx.UnderIlstMeta = true

				_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMeta(), Context: ctx})
				if err != nil {
					return err
				}

				_, err = mp4.Marshal(w, &mp4.Meta{}, ctx)
				if err != nil {
					return err
				}

				{ // hdlr
					_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeHdlr(), Context: ctx})
					if err != nil {
						return err
					}

					_, err = mp4.Marshal(w, &mp4.Hdlr{
						HandlerType: [4]byte{'m', 'd', 'i', 'r'},
						Reserved:    [3]uint32{0x6170706c, 0, 0},
					}, ctx)
					if err != nil {
						return err
					}

					_, err = w.EndBox()
					if err != nil {
						return err
					}
				}

				{ // ilst
					ctx.UnderIlst = true

					_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeIlst(), Context: ctx})
					if err != nil {
						return err
					}

					marshalData := func(val interface{}) error {
						_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeData()})
						if err != nil {
							return err
						}

						var boxData mp4.Data
						switch v := val.(type) {
						case string:
							boxData.DataType = mp4.DataTypeStringUTF8
							boxData.Data = []byte(v)
						case uint8:
							boxData.DataType = mp4.DataTypeSignedIntBigEndian
							boxData.Data = []byte{v}
						case uint32:
							boxData.DataType = mp4.DataTypeSignedIntBigEndian
							boxData.Data = make([]byte, 4)
							binary.BigEndian.PutUint32(boxData.Data, v)
						case []byte:
							boxData.DataType = mp4.DataTypeBinary
							boxData.Data = v
						default:
							panic("unsupported value")
						}

						_, err = mp4.Marshal(w, &boxData, ctx)
						if err != nil {
							return err
						}

						_, err = w.EndBox()
						return err
					}

					addMeta := func(tag mp4.BoxType, val interface{}) error {
						_, err = w.StartBox(&mp4.BoxInfo{Type: tag})
						if err != nil {
							return err
						}

						err = marshalData(val)
						if err != nil {
							return err
						}

						_, err = w.EndBox()
						return err
					}

					addExtendedMeta := func(name string, val interface{}) error {
						ctx.UnderIlstFreeMeta = true

						_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxType{'-', '-', '-', '-'}, Context: ctx})
						if err != nil {
							return err
						}

						{
							_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxType{'m', 'e', 'a', 'n'}, Context: ctx})
							if err != nil {
								return err
							}

							_, err = w.Write([]byte{0, 0, 0, 0})
							if err != nil {
								return err
							}

							_, err = io.WriteString(w, "com.apple.iTunes")
							if err != nil {
								return err
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						{
							_, err = w.StartBox(&mp4.BoxInfo{Type: mp4.BoxType{'n', 'a', 'm', 'e'}, Context: ctx})
							if err != nil {
								return err
							}

							_, err = w.Write([]byte{0, 0, 0, 0})
							if err != nil {
								return err
							}

							_, err = io.WriteString(w, name)
							if err != nil {
								return err
							}

							_, err = w.EndBox()
							if err != nil {
								return err
							}
						}

						err = marshalData(val)
						if err != nil {
							return err
						}

						ctx.UnderIlstFreeMeta = false

						_, err = w.EndBox()
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'n', 'a', 'm'}, meta.Data[0].Relationships.Tracks.Data[index].Attributes.Name)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'a', 'l', 'b'}, meta.Data[0].Attributes.Name)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'A', 'R', 'T'}, meta.Data[0].Relationships.Tracks.Data[index].Attributes.ArtistName)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'w', 'r', 't'}, meta.Data[0].Relationships.Tracks.Data[index].Attributes.ComposerName)
					if err != nil {
						return err
					}

					err = addMeta(mp4.BoxType{'\251', 'd', 'a', 'y'}, strings.Split(meta.Data[0].Attributes.ReleaseDate, "-")[0])
					if err != nil {
						return err
					}

					// cnID, err := strconv.ParseUint(meta.Data[0].Relationships.Tracks.Data[index].ID, 10, 32)
					// if err != nil {
					// 	return err
					// }

					// err = addMeta(mp4.BoxType{'c', 'n', 'I', 'D'}, uint32(cnID))
					// if err != nil {
					// 	return err
					// }

					err = addExtendedMeta("ISRC", meta.Data[0].Relationships.Tracks.Data[index].Attributes.Isrc)
					if err != nil {
						return err
					}

					if len(meta.Data[0].Relationships.Tracks.Data[index].Attributes.GenreNames) > 0 {
						err = addMeta(mp4.BoxType{'\251', 'g', 'e', 'n'}, meta.Data[0].Relationships.Tracks.Data[index].Attributes.GenreNames[0])
						if err != nil {
							return err
						}
					}

					if len(meta.Data) > 0 {
						album := meta.Data[0]

						err = addMeta(mp4.BoxType{'a', 'A', 'R', 'T'}, album.Attributes.ArtistName)
						if err != nil {
							return err
						}

						err = addMeta(mp4.BoxType{'c', 'p', 'r', 't'}, album.Attributes.Copyright)
						if err != nil {
							return err
						}

						var isCpil uint8
						if album.Attributes.IsCompilation {
							isCpil = 1
						}
						err = addMeta(mp4.BoxType{'c', 'p', 'i', 'l'}, isCpil)
						if err != nil {
							return err
						}

						err = addExtendedMeta("LABEL", album.Attributes.RecordLabel)
						if err != nil {
							return err
						}

						err = addExtendedMeta("UPC", album.Attributes.Upc)
						if err != nil {
							return err
						}

						// plID, err := strconv.ParseUint(album.ID, 10, 32)
						// if err != nil {
						// 	return err
						// }

						// err = addMeta(mp4.BoxType{'p', 'l', 'I', 'D'}, uint32(plID))
						// if err != nil {
						// 	return err
						// }
					}

					// if len(meta.Data[0].Relationships.Artists.Data) > 0 {
					// 	atID, err := strconv.ParseUint(meta.Data[0].Relationships.Artists.Data[index].ID, 10, 32)
					// 	if err != nil {
					// 		return err
					// 	}

					// 	err = addMeta(mp4.BoxType{'a', 't', 'I', 'D'}, uint32(atID))
					// 	if err != nil {
					// 		return err
					// 	}
					// }

					trkn := make([]byte, 8)
					binary.BigEndian.PutUint32(trkn, uint32(trackNum))
					binary.BigEndian.PutUint16(trkn[4:], uint16(trackTotal))
					err = addMeta(mp4.BoxType{'t', 'r', 'k', 'n'}, trkn)
					if err != nil {
						return err
					}

					// disk := make([]byte, 8)
					// binary.BigEndian.PutUint32(disk, uint32(meta.Attributes.DiscNumber))
					// err = addMeta(mp4.BoxType{'d', 'i', 's', 'k'}, disk)
					// if err != nil {
					// 	return err
					// }

					ctx.UnderIlst = false

					_, err = w.EndBox()
					if err != nil {
						return err
					}
				}

				ctx.UnderIlstMeta = false
				_, err = w.EndBox()
				if err != nil {
					return err
				}
			}

			ctx.UnderUdta = false
			_, err = w.EndBox()
			if err != nil {
				return err
			}
		}

		_, err = w.EndBox()
		if err != nil {
			return err
		}
	}

	{
		box, err := w.StartBox(&mp4.BoxInfo{Type: mp4.BoxTypeMdat()})
		if err != nil {
			return err
		}

		_, err = mp4.Marshal(w, &mp4.Mdat{Data: data}, box.Context)
		if err != nil {
			return err
		}

		mdat, err := w.EndBox()

		if err != nil {
			return err
		}

		var realStco mp4.Stco

		offset := mdat.Offset + mdat.HeaderSize
		for i := uint32(0); i < numSamples; i++ {
			if i%chunkSize == 0 {
				realStco.EntryCount++
				realStco.ChunkOffset = append(realStco.ChunkOffset, uint32(offset))
			}
			offset += uint64(len(info.samples[i].data))
		}

		_, err = stco.SeekToPayload(w)
		if err != nil {
			return err
		}
		_, err = mp4.Marshal(w, &realStco, box.Context)
		if err != nil {
			return err
		}
	}

	return nil
}

func decryptSong(port uint, info *SongInfo, keys []string, manifest *AutoGenerated, filename string, trackNum, trackTotal int) error {
	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.Dial("tcp", listenAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	var decrypted []byte
	var lastIndex uint32 = math.MaxUint8

	for _, sp := range info.samples {
		if lastIndex != sp.descIndex {
			if len(decrypted) != 0 {
				_, err := conn.Write([]byte{0, 0, 0, 0})
				if err != nil {
					return err
				}
			}
			keyUri := keys[sp.descIndex]
			id := manifest.Data[0].Relationships.Tracks.Data[trackNum-1].ID
			if keyUri == prefetchKey {
				id = defaultId
			}

			_, err := conn.Write([]byte{byte(len(id))})
			if err != nil {
				return err
			}
			_, err = io.WriteString(conn, id)
			if err != nil {
				return err
			}

			_, err = conn.Write([]byte{byte(len(keyUri))})
			if err != nil {
				return err
			}
			_, err = io.WriteString(conn, keyUri)
			if err != nil {
				return err
			}
		}
		lastIndex = sp.descIndex

		err := binary.Write(conn, binary.LittleEndian, uint32(len(sp.data)))
		if err != nil {
			return err
		}

		_, err = conn.Write(sp.data)
		if err != nil {
			return err
		}

		de := make([]byte, len(sp.data))
		_, err = io.ReadFull(conn, de)
		if err != nil {
			return err
		}

		decrypted = append(decrypted, de...)
	}
	_, _ = conn.Write([]byte{0, 0, 0, 0, 0})

	create, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer create.Close()

	return writeM4a(mp4.NewWriter(create), info, manifest, decrypted, trackNum, trackTotal)
}

func getMeta(albumId string, token string, storefront string) (*AutoGenerated, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/albums/%s", storefront, albumId), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Origin", "https://music.apple.com")
	query := url.Values{}
	query.Set("omit[resource]", "autos")
	query.Set("include", "tracks,artists,record-labels")
	query.Set("include[songs]", "artists")
	query.Set("fields[artists]", "name")
	query.Set("fields[albums:albums]", "artistName,artwork,name,releaseDate,url")
	query.Set("fields[record-labels]", "name")
	// query.Set("l", "en-gb")
	req.URL.RawQuery = query.Encode()
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return nil, errors.New(do.Status)
	}
	obj := new(AutoGenerated)
	err = json.NewDecoder(do.Body).Decode(&obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func writeCover(sanAlbumFolder, url string) error {
	covPath := filepath.Join(sanAlbumFolder, "cover.jpg")
	exists, err := fileExists(covPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	url = strings.Replace(url, "{w}x{h}", "1200x12000", 1)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	do, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return errors.New(do.Status)
	}
	f, err := os.Create(covPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, do.Body)
	if err != nil {
		return err
	}
	return nil
}

func Rip(albumId string, token string, storefront string, port uint, dir string) error {
	meta, err := getMeta(albumId, token, storefront)

	if err != nil {
		return err
	}

	albumFolder := fmt.Sprintf("%s - %s", meta.Data[0].Attributes.ArtistName, meta.Data[0].Attributes.Name)
	sanAlbumFolder := filepath.Join(dir, forbiddenNames.ReplaceAllString(albumFolder, "_"))

	os.MkdirAll(sanAlbumFolder, os.ModePerm)
	fmt.Println(albumFolder)

	_ = writeCover(sanAlbumFolder, meta.Data[0].Attributes.Artwork.URL)

	trackTotal := len(meta.Data[0].Relationships.Tracks.Data)

	for trackNum, track := range meta.Data[0].Relationships.Tracks.Data {
		trackNum++
		manifest, err := getInfoFromAdam(track.ID, token, storefront)
		if err != nil {
			continue
		}
		if manifest.Attributes.ExtendedAssetUrls.EnhancedHls == "" {
			continue
		}

		filename := fmt.Sprintf("%02d. %s.m4a", trackNum, forbiddenNames.ReplaceAllString(track.Attributes.Name, "_"))

		trackPath := filepath.Join(sanAlbumFolder, filename)

		if exists, err := fileExists(trackPath); !exists && err == nil {
			trackUrl, keys, err := extractMedia(manifest.Attributes.ExtendedAssetUrls.EnhancedHls)
			if err != nil {
				continue
			}

			info, err := extractSong(trackUrl)
			if err != nil {
				continue
			}

			samplesOk := true
			for samplesOk {
				for _, i := range info.samples {
					if int(i.descIndex) >= len(keys) {
						samplesOk = false
					}
				}
				break
			}
			if !samplesOk {
				continue
			}
			err = decryptSong(port, info, keys, meta, trackPath, trackNum, trackTotal)
			if err != nil {
				continue
			}
		}
	}
	return err
}

func extractMedia(b string) (string, []string, error) {
	masterUrl, err := url.Parse(b)
	if err != nil {
		return "", nil, err
	}
	resp, err := http.Get(b)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, errors.New(resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	masterString := string(body)
	from, listType, err := m3u8.DecodeFrom(strings.NewReader(masterString), true)
	if err != nil || listType != m3u8.MASTER {
		return "", nil, errors.New("m3u8 not of master type")
	}
	master := from.(*m3u8.MasterPlaylist)
	var streamUrl *url.URL
	sort.Slice(master.Variants, func(i, j int) bool {
		return master.Variants[i].AverageBandwidth > master.Variants[j].AverageBandwidth
	})
	for _, variant := range master.Variants {
		if variant.Codecs == "alac" {
			streamUrlTemp, err := masterUrl.Parse(variant.URI)
			if err != nil {
				return "", nil, err
			}
			streamUrl = streamUrlTemp
			break
		}
	}
	if streamUrl == nil {
		return "", nil, errors.New("no alac codec found")
	}
	var keys []string
	keys = append(keys, prefetchKey)
	streamUrl.Path = strings.TrimSuffix(streamUrl.Path, ".m3u8") + "_m.mp4"
	regex := regexp.MustCompile(`"(skd?://[^"]*)"`)
	matches := regex.FindAllStringSubmatch(masterString, -1)
	for _, match := range matches {
		if strings.HasSuffix(match[1], "c23") || strings.HasSuffix(match[1], "c6") {
			keys = append(keys, match[1])
		}
	}
	return streamUrl.String(), keys, nil
}

func extractSong(url string) (*SongInfo, error) {
	track, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer track.Body.Close()
	if track.StatusCode != http.StatusOK {
		return nil, errors.New(track.Status)
	}
	rawSong, err := io.ReadAll(track.Body)
	if err != nil {
		return nil, err
	}

	f := bytes.NewReader(rawSong)

	trex, err := mp4.ExtractBoxWithPayload(f, nil, []mp4.BoxType{
		mp4.BoxTypeMoov(),
		mp4.BoxTypeMvex(),
		mp4.BoxTypeTrex(),
	})
	if err != nil || len(trex) != 1 {
		return nil, err
	}
	trexPay := trex[0].Payload.(*mp4.Trex)

	stbl, err := mp4.ExtractBox(f, nil, []mp4.BoxType{
		mp4.BoxTypeMoov(),
		mp4.BoxTypeTrak(),
		mp4.BoxTypeMdia(),
		mp4.BoxTypeMinf(),
		mp4.BoxTypeStbl(),
	})
	if err != nil || len(stbl) != 1 {
		return nil, err
	}

	enca, err := mp4.ExtractBoxWithPayload(f, stbl[0], []mp4.BoxType{
		mp4.BoxTypeStsd(),
		mp4.BoxTypeEnca(),
	})
	if err != nil {
		return nil, err
	}

	aalac, err := mp4.ExtractBoxWithPayload(f, &enca[0].Info,
		[]mp4.BoxType{BoxTypeAlac()})
	if err != nil || len(aalac) != 1 {
		return nil, err
	}

	extracted := &SongInfo{
		r:         f,
		alacParam: aalac[0].Payload.(*Alac),
	}

	moofs, err := mp4.ExtractBox(f, nil, []mp4.BoxType{
		mp4.BoxTypeMoof(),
	})
	if err != nil || len(moofs) <= 0 {
		return nil, err
	}

	mdats, err := mp4.ExtractBoxWithPayload(f, nil, []mp4.BoxType{
		mp4.BoxTypeMdat(),
	})
	if err != nil || len(mdats) != len(moofs) {
		return nil, err
	}

	for i, moof := range moofs {
		tfhd, err := mp4.ExtractBoxWithPayload(f, moof, []mp4.BoxType{
			mp4.BoxTypeTraf(),
			mp4.BoxTypeTfhd(),
		})
		if err != nil || len(tfhd) != 1 {
			return nil, err
		}
		tfhdPay := tfhd[0].Payload.(*mp4.Tfhd)
		index := tfhdPay.SampleDescriptionIndex
		if index != 0 {
			index--
		}

		truns, err := mp4.ExtractBoxWithPayload(f, moof, []mp4.BoxType{
			mp4.BoxTypeTraf(),
			mp4.BoxTypeTrun(),
		})
		if err != nil || len(truns) <= 0 {
			return nil, err
		}

		mdat := mdats[i].Payload.(*mp4.Mdat).Data
		for _, t := range truns {
			for _, en := range t.Payload.(*mp4.Trun).Entries {
				info := SampleInfo{descIndex: index}

				switch {
				case t.Payload.CheckFlag(0x200):
					info.data = mdat[:en.SampleSize]
					mdat = mdat[en.SampleSize:]
				case tfhdPay.CheckFlag(0x10):
					info.data = mdat[:tfhdPay.DefaultSampleSize]
					mdat = mdat[tfhdPay.DefaultSampleSize:]
				default:
					info.data = mdat[:trexPay.DefaultSampleSize]
					mdat = mdat[trexPay.DefaultSampleSize:]
				}

				switch {
				case t.Payload.CheckFlag(0x100):
					info.duration = en.SampleDuration
				case tfhdPay.CheckFlag(0x8):
					info.duration = tfhdPay.DefaultSampleDuration
				default:
					info.duration = trexPay.DefaultSampleDuration
				}

				extracted.samples = append(extracted.samples, info)
			}
		}
		if len(mdat) != 0 {
			return nil, errors.New("offset mismatch")
		}
	}

	return extracted, nil
}

func init() {
	mp4.AddBoxDef((*Alac)(nil))
}

func BoxTypeAlac() mp4.BoxType { return mp4.StrToBoxType("alac") }

func getInfoFromAdam(adamId string, token string, storefront string) (*SongData, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("https://amp-api.music.apple.com/v1/catalog/%s/songs/%s", storefront, adamId), nil)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("extend", "extendedAssetUrls")
	query.Set("include", "albums")
	request.URL.RawQuery = query.Encode()

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	request.Header.Set("User-Agent", "iTunes/12.11.3 (Windows; Microsoft Windows 10 x64 Professional Edition (Build 19041); x64) AppleWebKit/7611.1022.4001.1 (dt:2)")
	request.Header.Set("Origin", "https://music.apple.com")

	do, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		return nil, errors.New(do.Status)
	}

	obj := new(ApiResult)
	err = json.NewDecoder(do.Body).Decode(&obj)
	if err != nil {
		return nil, err
	}

	for _, d := range obj.Data {
		if d.ID == adamId {
			return &d, nil
		}
	}
	return nil, nil
}

func getToken() (string, error) {
	req, err := http.NewRequest("GET", "https://beta.music.apple.com", nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	regex := regexp.MustCompile(`/assets/index-legacy-[^/]+\.js`)
	indexJsUri := regex.FindString(string(body))

	req, err = http.NewRequest("GET", "https://beta.music.apple.com"+indexJsUri, nil)
	if err != nil {
		return "", err
	}

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	regex = regexp.MustCompile(`eyJh([^"]*)`)
	token := regex.FindString(string(body))

	return token, nil
}
