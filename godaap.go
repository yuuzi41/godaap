package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/tcolgate/mp3"
	bolt "go.etcd.io/bbolt"

	"encoding/gob"
	"net/http"

	"github.com/yuuzi41/godaap/dummylistener"
)

func GenerateDmap(idata interface{}) []byte {
	switch tdata := idata.(type) {
	case bool:
		if tdata {
			return []byte{1}
		} else {
			return []byte{0}
		}
	case uint8:
		return []byte{tdata}
	case uint16:
		r := make([]byte, 2)
		binary.BigEndian.PutUint16(r, tdata)
		return r
	case uint32:
		r := make([]byte, 4)
		binary.BigEndian.PutUint32(r, tdata)
		return r
	case uint64:
		r := make([]byte, 8)
		binary.BigEndian.PutUint64(r, tdata)
		return r
	case string:
		return []byte(tdata)
	case []byte:
		return tdata
	case map[string]interface{}:
		buf := new(bytes.Buffer)
		for k, v := range tdata {
			slice := GenerateDmap(v)
			buf.WriteString(k)
			binary.Write(buf, binary.BigEndian, uint32(len(slice)))
			buf.Write(slice)
		}
		return buf.Bytes()
	case []map[string]interface{}:
		buf := new(bytes.Buffer)
		for _, v := range tdata {
			slice := GenerateDmap(v)
			buf.Write(slice)
		}
		return buf.Bytes()
	default:
		log.Println("unreconizable type", idata)
	}
	return []byte{}
}

func generateResponse() []byte {
	return []byte{}
}

func tempServerInfo() []byte {
	return GenerateDmap(map[string]interface{}{
		"msrv": map[string]interface{}{
			"mstt": uint32(200),
			"mpro": []byte{0x00, 0x02, 0x00, 0x0a},
			"apro": []byte{0x00, 0x03, 0x00, 0x0d},
			//"aeSV": []byte{0x00, 0x03, 0x00, 0x0d},
			"minm": "test",
			"mslr": false,
			"msau": uint8(0),
			//"mslr": true,
			//"msau": uint8(2),
			"mstm": uint32(1800),
			"msed": false,
			"msex": true,
			"mspi": true,
			"msix": true,
			"msbr": true,
			"msqy": true,
			"msup": true,
			//"msrs": false,
			"msdc": uint32(1),
			"msal": false,
			"ated": uint16(7),
			"asgr": uint16(0),
		},
	})
}

func tempLogin() []byte {
	return GenerateDmap(map[string]interface{}{
		"mlog": map[string]interface{}{
			"mstt": uint32(200),
			"mlid": uint32(1), //session-id, it seems meaningless
		},
	})
}

func tempUpdate() []byte {
	return GenerateDmap(map[string]interface{}{
		"mupd": map[string]interface{}{
			"mstt": uint32(200),
			"musr": uint32(3), //todo: database version
		},
	})
}

func tempDatabases() []byte {
	var songcount int = 0
	boltDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("songs"))
		if b != nil {
			songcount = b.Stats().KeyN
		}
		return nil
	})
	return GenerateDmap(map[string]interface{}{
		"avdb": []map[string]interface{}{
			{"mstt": uint32(200)},
			{"muty": uint8(0)},
			{"mtco": uint32(1)},
			{"mrco": uint32(1)},
			{
				"mlcl": []map[string]interface{}{
					{
						"mlit": map[string]interface{}{
							"miid": uint32(1),
							"mper": uint64(1),
							"mdbk": uint32(1),
							"minm": "test",
							"mimc": uint32(songcount), //count of songs
							"mctc": uint32(1),
							"meds": uint32(3),
						},
					},
				},
			},
		},
	})
}

func tempDatabaseItems() []byte {
	var musiclist []map[string]interface{} = nil
	/*
		musiclist = []map[string]interface{}{
			{
				"mlit": map[string]interface{}{
					"mikd": uint8(2),          //kind
					"miid": uint32(10),        //id
					"mper": uint64(10),        //id
					"asdk": uint8(0),          //datakind
					"aeMK": uint8(1),          //mediakind
					"aeMk": uint8(1),          //mediakind
					"minm": "ABC",             //title
					"asal": "Hoge",            //album
					"asar": "Fuga",            //artist
					"asgn": "Trance",          //genre
					"astm": uint32(10 * 1000), //time
					"astn": uint16(1),         //track
					"astc": uint16(2),         //track count
					"assz": uint32(1000),      //size
					"asfm": "wav",             //format
					"asbr": uint16(1411),
				},
			},
			{
				"mlit": map[string]interface{}{
					"mikd": uint8(2),          //kind
					"miid": uint32(11),        //id
					"mper": uint64(11),        //id
					"asdk": uint8(0),          //datakind
					"aeMK": uint8(1),          //mediakind
					"aeMk": uint8(1),          //mediakind
					"minm": "DEF",             //title
					"asal": "Hoge",            //album
					"asar": "Fuga",            //artist
					"asgn": "Trance",          //genre
					"astm": uint32(10 * 1000), //time
					"astn": uint16(2),         //track
					"astc": uint16(2),         //track count
					"assz": uint32(1000),      //size
					"asfm": "m4a",             //format
					"asbr": uint16(320),
					"ased": uint16(1),
					"asac": uint16(1),
				},
			},
		}
	*/
	boltDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("songs"))
		if b != nil {
			musiclist = make([]map[string]interface{}, 0, b.Stats().KeyN)

			return b.ForEach(func(k, v []byte) error {
				var md Metadata
				buf := bytes.NewBuffer(v)
				_ = gob.NewDecoder(buf).Decode(&md)
				id := binary.BigEndian.Uint32(k)

				//dmap.itemkind,dmap.itemid,daap.songalbum,daap.songartist,daap.songgenre,daap.songtime,daap.songtracknumber,daap.songformat
				musiclist = append(musiclist, map[string]interface{}{
					"mlit": map[string]interface{}{
						"mikd": uint8(2),               //kind
						"miid": uint32(id),             //id
						"mper": uint64(id),             //id
						"asdk": uint8(0),               //datakind
						"aeMK": uint8(1),               //mediakind
						"aeMk": uint8(1),               //mediakind
						"minm": md.Title,               //title
						"asal": md.Album,               //album
						"asar": md.Artist,              //artist
						"asaa": md.AlbumArtist,         //albumartist
						"asgn": md.Genre,               //genre
						"ascp": md.Composer,            //composer
						"asyr": uint16(md.Year),        //year
						"astm": uint32(md.Time),        //time
						"astn": uint16(md.TrackNumber), //track
						"astc": uint16(md.TrackCount),  //track count
						"asdn": uint16(md.DiscNumber),  //disc
						"asdc": uint16(md.DiscCount),   //disc count
						"assz": uint32(md.Length),      //size
						"asfm": md.Format,              //format
						"asbr": uint16(md.BitRate),
						"ased": uint16(1),
						"asac": uint16(1),
					},
				})
				return nil
			})
		}

		return nil
	})

	return GenerateDmap(map[string]interface{}{
		"adbs": []map[string]interface{}{
			{"mstt": uint32(200)},
			{"muty": uint8(0)},
			{"mtco": uint32(len(musiclist))},
			{"mrco": uint32(len(musiclist))},
			{"mlcl": musiclist},
		},
	})
}

func tempContainers() []byte {
	return GenerateDmap(map[string]interface{}{
		"aply": []map[string]interface{}{
			{"mstt": uint32(200)},
			{"muty": uint8(0)},
			{"mtco": uint32(1)},
			{"mrco": uint32(1)},
			{
				"mlcl": map[string]interface{}{
					"mlit": map[string]interface{}{
						"miid": uint32(1),
						"mper": uint64(1),
						"minm": "test",
						//"mpco": uint32(0),
						//"aePP": false,
						//"aePS": false,
						//"aeSP": false,
						//"aeSG": false,
						"abpl": true,
						"mimc": uint32(2),
					},
				},
			},
		},
	})
}

func tempContainerItems() []byte {
	var musiclist []map[string]interface{} = nil
	boltDb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("songs"))
		if b != nil {
			musiclist = make([]map[string]interface{}, 0, b.Stats().KeyN)

			return b.ForEach(func(k, v []byte) error {
				id := binary.BigEndian.Uint32(k)

				musiclist = append(musiclist, map[string]interface{}{
					"mlit": map[string]interface{}{
						"miid": uint32(id), //id
						"mcti": uint32(id), //id
					},
				})
				return nil
			})
		}

		return nil
	})

	return GenerateDmap(map[string]interface{}{
		"apso": []map[string]interface{}{
			{"mstt": uint32(200)},
			{"muty": uint8(0)},
			{"mtco": uint32(len(musiclist))},
			{"mrco": uint32(len(musiclist))},
			{"mlcl": musiclist},
		},
	})
}

var boltDb *bolt.DB

func init() {
	var err error
	boltDb, err = bolt.Open("test.db", 0644, nil)
	if err != nil {
		panic(err)
	}
	boltDb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("songs"))
		if err != nil {
			return (err)
		}

		_, err = tx.CreateBucketIfNotExists([]byte("artworks"))
		if err != nil {
			return (err)
		}

		b, err := tx.CreateBucket([]byte("info"))
		// if err2 is error, it means that the bucket is already here and we don't need to init values
		if err == nil {
			r := make([]byte, 4)
			binary.BigEndian.PutUint32(r, 2)
			b.Put([]byte("version"), r)

			rand.Seed(time.Now().UnixNano())
			r = make([]byte, 8)
			binary.BigEndian.PutUint64(r, rand.Uint64())
			b.Put([]byte("databaseid"), r)
		}
		return nil
	})
}

type Metadata struct {
	Format      string
	Title       string
	Album       string
	Artist      string
	AlbumArtist string
	Composer    string
	Genre       string
	Year        int
	TrackNumber int
	TrackCount  int
	DiscNumber  int
	DiscCount   int
	Lyrics      string
	Comment     string
	ArtworkMime string

	Time    int //millisecond
	Length  int
	BitRate int

	Path string
}

// ScanFile returns the offset of the first occurence of a []byte in a file,
// or -1 if []byte was not found in file, and seeks to the beginning of the searched []byte
func SeekFile(f io.ReadSeeker, search []byte) int64 {
	ix := 0
	r := bufio.NewReader(f)
	offset := int64(0)
	for ix < len(search) {
		b, err := r.ReadByte()
		if err != nil {
			return -1
		}
		if search[ix] == b {
			ix++
		} else {
			ix = 0
		}
		offset++
	}
	f.Seek(offset, 0) // Seeks to the beginning of the searched []byte
	return offset - int64(len(search))
}

func scan(scanpath string) {
	log.Println("Start Scan")
	boltDb.Batch(func(tx *bolt.Tx) error {
		b1 := tx.Bucket([]byte("songs"))
		b2 := tx.Bucket([]byte("artworks"))
		if b1 != nil && b2 != nil {
			filepath.Walk(scanpath, func(path string, info os.FileInfo, err error) error {
				if info.IsDir() {
					// ignore directories
					return nil
				}

				musicfileAbs, err := filepath.Abs(path)
				if err != nil {
					return nil
				}

				fp, err := os.Open(musicfileAbs)
				if err != nil {
					panic(err)
				}
				defer fp.Close()

				m, err := tag.ReadFrom(fp)
				if err != nil {
					log.Fatal(err)
				}

				// calc music duration
				duration := 0.0
				if m.Format() == "MP4" {
					//todo
					SeekFile(fp, []byte("mvhd"))
				} else if m.FileType() == "MP3" {
					d := mp3.NewDecoder(fp)
					var f mp3.Frame
					skipped := 0

					for {
						if err := d.Decode(&f, &skipped); err != nil {
							if err == io.EOF {
								break
							}
						}

						duration = duration + f.Duration().Seconds()
					}
				}

				log.Printf("Scan : Title %s\n", m.Title())

				tn, tc := m.Track()
				dn, dc := m.Disc()

				pic := m.Picture()
				picMIME := ""
				if pic != nil {
					picMIME = pic.MIMEType
				}

				md := Metadata{
					Format:      string(m.FileType()),
					Title:       m.Title(),
					Album:       m.Album(),
					Artist:      m.Artist(),
					AlbumArtist: m.AlbumArtist(),
					Composer:    m.Composer(),
					Genre:       m.Genre(),
					Year:        m.Year(),
					TrackNumber: tn,
					TrackCount:  tc,
					DiscNumber:  dn,
					DiscCount:   dc,
					Lyrics:      m.Lyrics(),
					Comment:     m.Comment(),
					Time:        int(duration * 1000),
					Length:      int(info.Size()),
					BitRate:     int(float64(info.Size()) * 8.0 / 1024.0 / duration),
					ArtworkMime: picMIME,
					Path:        musicfileAbs,
				}

				// calc id
				hasher := fnv.New32()
				hasher.Write([]byte(musicfileAbs))
				songid := hasher.Sum(nil)

				buf := bytes.NewBuffer(nil)
				err = gob.NewEncoder(buf).Encode(md)
				if err == nil {
					b1.Put(songid, buf.Bytes())
				}

				if pic != nil {
					b2.Put(songid, pic.Data)
				}

				return nil
			})
		}
		return nil
	})
	log.Println("End Scan")
}

func main() {
	scan("./music")

	r := gin.Default()
	auth := func(c *gin.Context) {
		if true {
			c.Next()
		} else {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(func(c *gin.Context) {
		c.Header("DAAP-Server", "godaap")
		c.Header("Ranges", "bytes")
	})
	r.GET("/server-info", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/x-dmap-tagged", tempServerInfo())
	})
	r.GET("/login", auth, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/x-dmap-tagged", tempLogin())
	})
	r.GET("/logout", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusOK)
	})
	r.GET("/update", auth, func(c *gin.Context) {
		deltaStr := c.Query("delta")
		delta, err := strconv.Atoi(deltaStr)
		if err == nil && delta > 0 {
			// i guess it might be needed because a client assumes something like Comet
			time.Sleep(30 * time.Second)
		}
		c.Data(http.StatusOK, "application/x-dmap-tagged", tempUpdate())
	})
	r.GET("/databases", auth, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/x-dmap-tagged", tempDatabases())
	})
	r.GET("/databases/1/items", auth, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/x-dmap-tagged", tempDatabaseItems())
	})
	r.GET("/databases/1/containers", auth, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/x-dmap-tagged", tempContainers())
	})
	r.GET("/databases/1/containers/1/items", auth, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/x-dmap-tagged", tempContainerItems())
	})

	r.GET("/databases/1/items/:song", func(c *gin.Context) {
		filename := c.Param("song")
		idStr := strings.TrimSuffix(filename, filepath.Ext(filename))
		id, err := strconv.Atoi(idStr)
		idBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(idBytes, uint32(id))

		if err == nil {
			err = boltDb.View(func(tx *bolt.Tx) error {
				b1 := tx.Bucket([]byte("songs"))
				if b1 != nil {
					mdbinary := b1.Get(idBytes)
					if mdbinary == nil {
						return fmt.Errorf("id %d is not found", id)
					} else {
						var md Metadata
						buf := bytes.NewBuffer(mdbinary)
						_ = gob.NewDecoder(buf).Decode(&md)

						c.File(md.Path)
						return nil
					}
				} else {
					return fmt.Errorf("song bucket is not found")
				}
			})

			if err != nil {
				c.AbortWithError(http.StatusNotFound, err)
			}
		} else {
			c.AbortWithError(http.StatusNotFound, err)
		}
	})

	r.GET("/databases/1/items/:song/extra_data/artwork", auth, func(c *gin.Context) {
		idStr := c.Param("song")
		id, err := strconv.Atoi(idStr)
		idBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(idBytes, uint32(id))

		if err == nil {
			err = boltDb.View(func(tx *bolt.Tx) error {
				b1 := tx.Bucket([]byte("songs"))
				b2 := tx.Bucket([]byte("artworks"))
				if b1 != nil && b2 != nil {
					artwork := b2.Get(idBytes)
					if artwork == nil {
						return fmt.Errorf("id %d is not found", id)
					} else {
						mdbinary := b1.Get(idBytes)
						var md Metadata
						buf := bytes.NewBuffer(mdbinary)
						_ = gob.NewDecoder(buf).Decode(&md)

						c.Data(http.StatusOK, md.ArtworkMime, artwork)
						return nil
					}
				} else {
					return fmt.Errorf("artwork bucket is not found")
				}
			})

			if err != nil {
				c.AbortWithError(http.StatusNotFound, err)
			}
		} else {
			c.AbortWithError(http.StatusNotFound, err)
		}
	})

	ln, err := net.Listen("tcp", ":3689")
	if err != nil {
		log.Panicln("Couldn't listen")
	}
	ln2, err := dummylistener.Listener(ln)
	if err != nil {
		log.Panicln("Couldn't listen")
	}
	r.RunListener(ln2)
}
