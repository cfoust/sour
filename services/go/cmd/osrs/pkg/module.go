// ../Runescape-Map-Viewer/rsconv.java
package pkg

import "fmt"
import "os"

type Buffer struct {
	*Cacheable
	payload         []byte
	CurrentPosition int
}

func NewBuffer(payload []byte) (rcvr *Buffer) {
	rcvr = &Buffer{}
	rcvr.payload = payload
	rcvr.CurrentPosition = 0
	return
}
func (rcvr *Buffer) GetUIncrementalSmart() (int) {
	value := 0
	var remainder int
	for remainder = rcvr.ReadUSmart(); remainder == 32767; remainder = rcvr.ReadUSmart() {
		value += 32767
	}
	value += remainder
	return value
}
func (rcvr *Buffer) Read24Int() (int) {
	rcvr.CurrentPosition += 3
	return payload[rcvr.CurrentPosition-3]&0xff<<16 + payload[rcvr.CurrentPosition-2]&0xff<<8 + payload[rcvr.CurrentPosition-1]&0xff
}
func (rcvr *Buffer) ReadInt() (int) {
	rcvr.CurrentPosition += 4
	return payload[rcvr.CurrentPosition-4]&0xff<<24 + payload[rcvr.CurrentPosition-3]&0xff<<16 + payload[rcvr.CurrentPosition-2]&0xff<<8 + payload[rcvr.CurrentPosition-1]&0xff
}

func (rcvr *Buffer) ReadSignedByte() (byte) {
	return payload[++rcvr.CurrentPosition]
}

func (rcvr *Buffer) ReadSmart() (int) {
	value := payload[rcvr.CurrentPosition] & 0xff
	if value < 128 {
		return rcvr.ReadUnsignedByte() - 64
	} else {
		return rcvr.ReadUShort() - 49152
	}
}
func (rcvr *Buffer) ReadString() (string) {
	index := rcvr.CurrentPosition
	for payload[++rcvr.CurrentPosition] != 10 {
		<<unimp_stmt[*grammar.JEmpty]>>
	}
	return NewString(rcvr.payload, index, rcvr.CurrentPosition-index-1)
}
func (rcvr *Buffer) ReadTriByte() (int) {
	rcvr.CurrentPosition += 3
	return payload[rcvr.CurrentPosition-3]&0xff<<16 + payload[rcvr.CurrentPosition-2]&0xff<<8 + payload[rcvr.CurrentPosition-1]&0xff
}
func (rcvr *Buffer) ReadUShort() (int) {
	rcvr.CurrentPosition += 2
	return payload[rcvr.CurrentPosition-2]&0xff<<8 + payload[rcvr.CurrentPosition-1]&0xff
}
func (rcvr *Buffer) ReadUSmart() (int) {
	value := payload[rcvr.CurrentPosition] & 0xff
	if value < 128 {
		return rcvr.ReadUnsignedByte()
	} else {
		return rcvr.ReadUShort() - 32768
	}
}
func (rcvr *Buffer) ReadUnsignedByte() (int) {
	return payload[++rcvr.CurrentPosition] & 0xff
}

type Cacheable struct {
	*Linkable
	NextCacheable     *Cacheable
	PreviousCacheable *Cacheable
}

func NewCacheable() (rcvr *Cacheable) {
	rcvr = &Cacheable{}
	return
}
func (rcvr *Cacheable) UnlinkCacheable() {
	if rcvr.PreviousCacheable == nil {
	} else {
		rcvr.PreviousCacheable.nextCacheable = rcvr.NextCacheable
		rcvr.NextCacheable.previousCacheable = rcvr.PreviousCacheable
		rcvr.NextCacheable = nil
		rcvr.PreviousCacheable = nil
	}
}

const CLIENT_NAME = "Map Viewer"
const WIDTH = 765
const HEIGHT = 503
const START_X = 3136
const START_Y = 3136
const BRIGHTNESS = 0.80000000000000004d
const CONFIG_CRC = 2
const UPDATE_CRC = 5
const TEXTURES_CRC = 6

var CACHE_DIRECTORY = fmt.Sprintf("%v%v", System.getProperty("user.home")+File.separator+"Map-Viewer", File.separator)

type Configuration struct {
}

func NewConfiguration() (rcvr *Configuration) {
	rcvr = &Configuration{}
	return
}

type Deque struct {
	head    *Linkable
	current *Linkable
}

func NewDeque() (rcvr *Deque) {
	rcvr = &Deque{}
	rcvr.head = NewLinkable()
	rcvr.head.previous = rcvr.head
	rcvr.head.next = rcvr.head
	return
}
func (rcvr *Deque) Clear() {
	if rcvr.head.previous == rcvr.head {
		return
	}
	for {
		node := rcvr.head.previous
		if node == rcvr.head {
			return
		}
		node.unlink()
		if !(true) {
			break
		}
	}
}
func (rcvr *Deque) InsertHead(linkable *Linkable) {
	if linkable.next != nil {
		linkable.unlink()
	}
	linkable.next = rcvr.head.next
	linkable.previous = rcvr.head
	linkable.next.previous = linkable
	linkable.previous.next = linkable
}
func (rcvr *Deque) PopHead() (*Linkable) {
	node := rcvr.head.previous
	if node == rcvr.head {
		return nil
	} else {
		node.unlink()
		return node
	}
}
func (rcvr *Deque) ReverseGetFirst() (*Linkable) {
	node := rcvr.head.previous
	if node == rcvr.head {
		rcvr.current = nil
		return nil
	} else {
		rcvr.current = node.previous
		return node
	}
}
func (rcvr *Deque) ReverseGetNext() (*Linkable) {
	node := rcvr.current
	if node == rcvr.head {
		rcvr.current = nil
		return nil
	} else {
		rcvr.current = node.previous
		return node
	}
}

type FileArchive struct {
	buffer         []byte
	entries        int
	identifiers    []int
	extractedSizes []int
	sizes          []int
	indices        []int
	extracted      bool
}

func NewFileArchive(data []byte) (rcvr *FileArchive) {
	rcvr = &FileArchive{}
	buffer := NewBuffer(data)
	decompressedLength := buffer.readTriByte()
	compressedLength := buffer.readTriByte()
	if compressedLength != decompressedLength {
		output := make([]byte, decompressedLength)
		BZip2Decompressor.decompress(output, decompressedLength, data, compressedLength, 6)
		rcvr.buffer = output
		buffer = NewBuffer(rcvr.buffer)
		rcvr.extracted = true
	} else {
		rcvr.buffer = data
		rcvr.extracted = false
	}
	rcvr.entries = buffer.readUShort()
	rcvr.identifiers = make([]int, rcvr.entries)
	rcvr.extractedSizes = make([]int, rcvr.entries)
	rcvr.sizes = make([]int, rcvr.entries)
	rcvr.indices = make([]int, rcvr.entries)
	offset := buffer.currentPosition + rcvr.entries*10
	for file := 0; file < rcvr.entries; file++ {
		identifiers[file] = buffer.readInt()
		extractedSizes[file] = buffer.readTriByte()
		sizes[file] = buffer.readTriByte()
		indices[file] = offset
		offset += sizes[file]
	}
	return
}
func Decode(file int, filestoreIndices []*FileStore) (*FileArchive) {
	buffer := nil
	if try() {
		if filestoreIndices[0] != nil {
			buffer = filestoreIndices[0].readFile(file)
		}
	} else if catch_Exception(exception) {
		exception.printStackTrace()
	}
	if buffer == nil {
		return nil
	}
	streamLoader := NewFileArchive(buffer)
	return streamLoader
}
func (rcvr *FileArchive) ReadFile(name string) (<<array>>) {
	output := nil
	hash := 0
	name = name.toUpperCase()
	for index := 0; index < name.length(); index++ {
		hash = hash*61 + name.charAt(index) - 32
	}
	for file := 0; file < rcvr.entries; file++ {
		if identifiers[file] == hash {
			if output == nil {
				output = make([]byte, extractedSizes[file])
			}
			if !rcvr.extracted {
				BZip2Decompressor.decompress(output, extractedSizes[file], rcvr.buffer, sizes[file], indices[file])
			} else {
				System.arraycopy(rcvr.buffer, indices[file], output, 0, extractedSizes[file])
			}
			return output
		}
	}
	return nil
}

var buffer = make([]byte, 520)

type FileStore struct {
	dataFile   *RandomAccessFile
	indexFile  *RandomAccessFile
	storeIndex int
}

func NewFileStore(data *RandomAccessFile, index *RandomAccessFile, storeIndex int) (rcvr *FileStore) {
	rcvr = &FileStore{}
	rcvr.storeIndex = storeIndex
	rcvr.dataFile = data
	rcvr.indexFile = index
	return
}
func (rcvr *FileStore) ReadFile(id int) (<<array>>) {
	if try() {
		rcvr.seek(rcvr.indexFile, id*6)
		for in := 0; read < 6; read += in {
			in = rcvr.indexFile.read(buffer, read, 6-read)
			if in == -1 {
				return nil
			}
		}
		size := buffer[0]&0xff<<16 + buffer[1]&0xff<<8 + buffer[2]&0xff
		sector := buffer[3]&0xff<<16 + buffer[4]&0xff<<8 + buffer[5]&0xff
		if sector <= 0 || sector.(int64) > rcvr.dataFile.length()/520 {
			return nil
		}
		buf := make([]byte, size)
		totalRead := 0
		for part := 0; totalRead < size; part++ {
			if sector == 0 {
				return nil
			}
			rcvr.seek(rcvr.dataFile, sector*520)
			unread := size - totalRead
			if unread > 512 {
				unread = 512
			}
			for in := 0; read < unread+8; read += in {
				in = rcvr.dataFile.read(buffer, read, unread+8-read)
				if in == -1 {
					return nil
				}
			}
			currentIndex := buffer[0]&0xff<<8 + buffer[1]&0xff
			currentPart := buffer[2]&0xff<<8 + buffer[3]&0xff
			nextSector := buffer[4]&0xff<<16 + buffer[5]&0xff<<8 + buffer[6]&0xff
			currentFile := buffer[7] & 0xff
			if currentIndex != id || currentPart != part || currentFile != rcvr.storeIndex {
				return nil
			}
			if nextSector < 0 || nextSector.(int64) > rcvr.dataFile.length()/520 {
				return nil
			}
			for i := 0; i < unread; i++ {
				buf[++totalRead] = buffer[i+8]
			}
			sector = nextSector
		}
		return buf
	} else if catch_IOException(_ex) {
		return nil
	}
}
func (rcvr *FileStore) seek(file *RandomAccessFile, position int) {
	if try() {
		file.seek(position)
	} else if catch_Exception(e) {
		e.printStackTrace()
	}
}

type FileUtils struct {
}

func NewFileUtils() (rcvr *FileUtils) {
	rcvr = &FileUtils{}
	return
}
func DecompressGzip(data []byte) (<<array>>) {
	if try() {
		if data == nil {
			return nil
		}
		gzipInputStream := NewGZIPInputStream(NewByteArrayInputStream(data))
		bos := NewByteArrayOutputStream()
		buf := make([]byte, 1024)
		var len int
		for (len = gzipInputStream.read(buf)) > 0 {
			bos.write(buf, 0, len)
		}
		gzipInputStream.close()
		bos.close()
		return bos.toByteArray()
	} else if catch_Exception(e) {
		e.printStackTrace()
		return nil
	}
}
func DecompressZip(zipFile string, outputFolder string, deleteAfter bool) {
	buffer := make([]byte, 1024)
	if try() {
		folder := NewFile(outputFolder)
		if !folder.exists() {
			folder.mkdir()
		}
		zis := NewZipInputStream(NewFileInputStream(zipFile))
		ze := zis.getNextEntry()
		for ze != nil {
			fileName := ze.getName()
			newFile := NewFile(fmt.Sprintf("%v%v%v", outputFolder, File.separator, fileName))
			NewFile(newFile.getParent()).mkdirs()
			fos := NewFileOutputStream(newFile)
			var len int
			for (len = zis.read(buffer)) > 0 {
				fos.write(buffer, 0, len)
			}
			fos.close()
			ze = zis.getNextEntry()
		}
		zis.closeEntry()
		zis.close()
		if deleteAfter {
			NewFile(zipFile).delete()
		}
	} else if catch_Exception(exception) {
		exception.printStackTrace()
	}
}
func Findcachedir() (string) {
	cacheDirectory := NewFile(Configuration.CACHE_DIRECTORY)
	if !cacheDirectory.exists() {
		cacheDirectory.mkdir()
	}
	return Configuration.CACHE_DIRECTORY
}

type GameObject struct {
	Renderable     *Renderable
	Uid            int
	Z              int
	X              int
	Y              int
	XLocLow        int
	XLocHigh       int
	YLocHigh       int
	YLocLow        int
	Rendered       int
	CameraDistance int
	TurnValue      int
	TileHeight     int
	Mask           byte
}

func NewGameObject() (rcvr *GameObject) {
	rcvr = &GameObject{}
	return
}

type GroundDecoration struct {
	Renderable *Renderable
	ZPos       int
	YPos       int
	XPos       int
	Uid        int
	Mask       byte
}

func NewGroundDecoration() (rcvr *GroundDecoration) {
	rcvr = &GroundDecoration{}
	return
}

type IndexedImage struct {
	*Rasterizer2D
	Palette       []int
	PalettePixels []byte
	Width         int
	height        int
	drawOffsetX   int
	drawOffsetY   int
	ResizeWidth   int
	resizeHeight  int
}

func NewIndexedImage(archive *FileArchive, value string, id int) (rcvr *IndexedImage) {
	rcvr = &IndexedImage{}
	buffer := NewBuffer(archive.readFile(fmt.Sprintf("%v%v", value, ".dat")))
	data := NewBuffer(archive.readFile("index.dat"))
	data.currentPosition = buffer.readUShort()
	rcvr.ResizeWidth = data.readUShort()
	rcvr.resizeHeight = data.readUShort()
	colorLength := data.readUnsignedByte()
	rcvr.Palette = make([]int, colorLength)
	for index := 0; index < colorLength-1; index++ {
		Palette[index+1] = data.readTriByte()
	}
	for index := 0; index < id; index++ {
		data.currentPosition += 2
		buffer.currentPosition += data.readUShort() * data.readUShort()
		data.currentPosition++
	}
	rcvr.drawOffsetX = data.readUnsignedByte()
	rcvr.drawOffsetY = data.readUnsignedByte()
	rcvr.Width = data.readUShort()
	rcvr.height = data.readUShort()
	type := data.readUnsignedByte()
	pixels := rcvr.Width * rcvr.height
	rcvr.PalettePixels = make([]byte, pixels)
	if type == 0 {
		for index := 0; index < pixels; index++ {
			PalettePixels[index] = buffer.readSignedByte()
		}
	} else if type == 1 {
		for x := 0; x < rcvr.Width; x++ {
			for y := 0; y < rcvr.height; y++ {
				PalettePixels[x+y*rcvr.Width] = buffer.readSignedByte()
			}
		}
	}
	return
}
func (rcvr *IndexedImage) Resize() {
	if rcvr.Width == rcvr.ResizeWidth && rcvr.height == rcvr.resizeHeight {
		return
	}
	raster := make([]byte, rcvr.ResizeWidth*rcvr.resizeHeight)
	i := 0
	for y := 0; y < rcvr.height; y++ {
		for x := 0; x < rcvr.Width; x++ {
			raster[x+rcvr.drawOffsetX+(y+rcvr.drawOffsetY)*rcvr.ResizeWidth] = raster[++i]
		}
	}
	rcvr.PalettePixels = raster
	rcvr.Width = rcvr.ResizeWidth
	rcvr.height = rcvr.resizeHeight
	rcvr.drawOffsetX = 0
	rcvr.drawOffsetY = 0
}

type Linkable struct {
	Key      int64
	Previous *Linkable
	Next     *Linkable
}

func NewLinkable() (rcvr *Linkable) {
	rcvr = &Linkable{}
	return
}
func (rcvr *Linkable) Unlink() {
	if rcvr.Next == nil {
	} else {
		rcvr.Next.previous = rcvr.Previous
		rcvr.Previous.next = rcvr.Next
		rcvr.Previous = nil
		rcvr.Next = nil
	}
}

type MapDefinition struct {
	areas            []int
	mapFiles         []int
	landscapes       []int
	filestoreIndices []*FileStore
}

func NewMapDefinition() (rcvr *MapDefinition) {
	rcvr = &MapDefinition{}
	return
}
func (rcvr *MapDefinition) GetMapIndex(regionX int, regionY int, type int) (int) {
	id := type<<8 + regionY
	for area := 0; area < len(rcvr.areas); area++ {
		if areas[area] == id {
			if regionX == 0 {
				return <<unimp_expr[*grammar.JConditionalExpr]>>
			} else {
				return <<unimp_expr[*grammar.JConditionalExpr]>>
			}
		}
	}
	return -1
}
func (rcvr *MapDefinition) GetModel(id int) (<<array>>) {
	if try() {
		return FileUtils.decompressGzip(filestoreIndices[1].readFile(id))
	} else if catch_Exception(exception) {
		exception.printStackTrace()
	}
	return nil
}
func (rcvr *MapDefinition) Initialize(archive *FileArchive, filestoreIndices []*FileStore) {
	data := archive.readFile("map_index")
	stream := NewBuffer(data)
	size := stream.readUShort()
	rcvr.areas = make([]int, size)
	rcvr.mapFiles = make([]int, size)
	rcvr.landscapes = make([]int, size)
	for index := 0; index < size; index++ {
		areas[index] = stream.readUShort()
		mapFiles[index] = stream.readUShort()
		landscapes[index] = stream.readUShort()
	}
	rcvr.filestoreIndices = filestoreIndices
}

var Sine []int
var Cosine []int
var modelHeaderCache []*ModelHeader
var resourceProvider *MapDefinition
var ABoolean1684 bool
var MouseX int
var MouseY int
var AnInt1687 int
var modelIntArray3 []int
var anIntArray1688 = make([]int, 1000)
var projected_vertex_x = make([]int, 8000)
var projected_vertex_y = make([]int, 8000)
var projected_vertex_z = make([]int, 8000)
var camera_vertex_y = make([]int, 8000)
var camera_vertex_x = make([]int, 8000)
var camera_vertex_z = make([]int, 8000)
var anIntArray1668 = make([]int, 8000)
var depthListIndices = make([]int, 3000)
var faceLists = make([]int, 1600, 512)
var anIntArray1673 = make([]int, 12)
var anIntArrayArray1674 = make([]int, 12, 2000)
var anIntArray1675 = make([]int, 2000)
var anIntArray1676 = make([]int, 2000)
var anIntArray1677 = make([]int, 12)
var hasAnEdgeToRestrict = make([]bool, 8000)
var outOfReach = make([]bool, 8000)
var anIntArray1678 = make([]int, 10)
var anIntArray1679 = make([]int, 10)
var anIntArray1680 = make([]int, 10)
var modelIntArray4 []int

type Model struct {
	*Renderable
	MaxVertexDistanceXZPlane int
	VertexX                  []int
	MaximumYVertex           int
	VertexY                  []int
	MinimumXVertex           int
	MinimumZVertex           int
	VerticeCount             int
	AlsoVertexNormals        []*VertexNormal
	MaximumXVertex           int
	VertexZ                  []int
	MaximumZVertex           int
	TriangleCount            int
	FacePointA               []int
	FacePointB               []int
	FacePointC               []int
	FaceDrawType             []int
	FaceGroups               []int
	VertexGroups             []int
	ItemDropHeight           int
	diagonal3DAboveOrigin    int
	fitsOnSingleTile         bool
	textureTriangleCount     int
	faceHslA                 []int
	faceHslB                 []int
	faceHslC                 []int
	verticesParticle         []int
	triangleColours          []int16
	texture                  []int16
	textureCoordinates       []byte
	faceAlpha                []int
	faceRenderPriorities     []byte
	facePriority             byte
	maxRenderDepth           int
	texturesFaceA            []int16
	texturesFaceB            []int16
	texturesFaceC            []int16
	vertexVSkin              []int
	triangleTSkin            []int
	textureType              []byte
}

func NewModel(contouredGround bool, delayShading bool, model *Model) (rcvr *Model) {
	rcvr = &Model{}
	rcvr.fitsOnSingleTile = false
	rcvr.VerticeCount = model.verticeCount
	rcvr.TriangleCount = model.triangleCount
	rcvr.textureTriangleCount = model.textureTriangleCount
	if contouredGround {
		rcvr.VertexY = make([]int, rcvr.VerticeCount)
		for index := 0; index < rcvr.VerticeCount; index++ {
			VertexY[index] = model[index]
		}
	} else {
		rcvr.VertexY = model.vertexY
	}
	if delayShading {
		rcvr.faceHslA = make([]int, rcvr.TriangleCount)
		rcvr.faceHslB = make([]int, rcvr.TriangleCount)
		rcvr.faceHslC = make([]int, rcvr.TriangleCount)
		for index := 0; index < rcvr.TriangleCount; index++ {
			faceHslA[index] = model[index]
			faceHslB[index] = model[index]
			faceHslC[index] = model[index]
		}
		rcvr.FaceDrawType = make([]int, rcvr.TriangleCount)
		if model.faceDrawType == nil {
			for index := 0; index < rcvr.TriangleCount; index++ {
				FaceDrawType[index] = 0
			}
		} else {
			for index := 0; index < rcvr.TriangleCount; index++ {
				FaceDrawType[index] = model[index]
			}
		}
		<<super>> = make([]*VertexNormal, rcvr.VerticeCount)
		for index := 0; index < rcvr.VerticeCount; index++ {
			vertexNormalPrimary := <<super>>[index] = NewVertexNormal()
			vertexNormalSecondary := model[index]
			vertexNormalPrimary.normalX = vertexNormalSecondary.normalX
			vertexNormalPrimary.normalY = vertexNormalSecondary.normalY
			vertexNormalPrimary.normalZ = vertexNormalSecondary.normalZ
			vertexNormalPrimary.magnitude = vertexNormalSecondary.magnitude
		}
		rcvr.AlsoVertexNormals = model.alsoVertexNormals
	} else {
		rcvr.faceHslA = model.faceHslA
		rcvr.faceHslB = model.faceHslB
		rcvr.faceHslC = model.faceHslC
		rcvr.FaceDrawType = model.faceDrawType
	}
	rcvr.verticesParticle = model.verticesParticle
	rcvr.VertexX = model.vertexX
	rcvr.VertexZ = model.vertexZ
	rcvr.triangleColours = model.triangleColours
	rcvr.faceAlpha = model.faceAlpha
	rcvr.faceRenderPriorities = model.faceRenderPriorities
	rcvr.facePriority = model.facePriority
	rcvr.FacePointA = model.facePointA
	rcvr.FacePointB = model.facePointB
	rcvr.FacePointC = model.facePointC
	rcvr.texturesFaceA = model.texturesFaceA
	rcvr.texturesFaceB = model.texturesFaceB
	rcvr.texturesFaceC = model.texturesFaceC
	<<super>> = model.modelBaseY
	rcvr.textureCoordinates = model.textureCoordinates
	rcvr.texture = model.texture
	rcvr.MaxVertexDistanceXZPlane = model.maxVertexDistanceXZPlane
	rcvr.diagonal3DAboveOrigin = model.diagonal3DAboveOrigin
	rcvr.maxRenderDepth = model.maxRenderDepth
	rcvr.MinimumXVertex = model.minimumXVertex
	rcvr.MaximumZVertex = model.maximumZVertex
	rcvr.MinimumZVertex = model.minimumZVertex
	rcvr.MaximumXVertex = model.maximumXVertex
	return
}
func NewModel2(length int, model_segments []*Model) (rcvr *Model) {
	rcvr = &Model{}
	if try() {
		rcvr.fitsOnSingleTile = false
		renderTypeFlag := false
		priorityFlag := false
		alphaFlag := false
		tSkinFlag := false
		colorFlag := false
		textureFlag := false
		coordinateFlag := false
		rcvr.VerticeCount = 0
		rcvr.TriangleCount = 0
		rcvr.textureTriangleCount = 0
		rcvr.facePriority = -1
		var build *Model
		for segment_index := 0; segment_index < length; segment_index++ {
			build = model_segments[segment_index]
			if build != nil {
				rcvr.VerticeCount += build.verticeCount
				rcvr.TriangleCount += build.triangleCount
				rcvr.textureTriangleCount += build.textureTriangleCount
				renderTypeFlag |= build.faceDrawType != nil
				alphaFlag |= build.faceAlpha != nil
				if build.faceRenderPriorities != nil {
					priorityFlag = true
				} else {
					if rcvr.facePriority == -1 {
						rcvr.facePriority = build.facePriority
					}
					if rcvr.facePriority != build.facePriority {
						priorityFlag = true
					}
				}
				tSkinFlag |= build.triangleTSkin != nil
				colorFlag |= build.triangleColours != nil
				textureFlag |= build.texture != nil
				coordinateFlag |= build.textureCoordinates != nil
			}
		}
		rcvr.verticesParticle = make([]int, rcvr.VerticeCount)
		rcvr.VertexX = make([]int, rcvr.VerticeCount)
		rcvr.VertexY = make([]int, rcvr.VerticeCount)
		rcvr.VertexZ = make([]int, rcvr.VerticeCount)
		rcvr.vertexVSkin = make([]int, rcvr.VerticeCount)
		rcvr.FacePointA = make([]int, rcvr.TriangleCount)
		rcvr.FacePointB = make([]int, rcvr.TriangleCount)
		rcvr.FacePointC = make([]int, rcvr.TriangleCount)
		if colorFlag {
			rcvr.triangleColours = make([]int16, rcvr.TriangleCount)
		}
		if renderTypeFlag {
			rcvr.FaceDrawType = make([]int, rcvr.TriangleCount)
		}
		if priorityFlag {
			rcvr.faceRenderPriorities = make([]byte, rcvr.TriangleCount)
		}
		if alphaFlag {
			rcvr.faceAlpha = make([]int, rcvr.TriangleCount)
		}
		if tSkinFlag {
			rcvr.triangleTSkin = make([]int, rcvr.TriangleCount)
		}
		if textureFlag {
			rcvr.texture = make([]int16, rcvr.TriangleCount)
		}
		if coordinateFlag {
			rcvr.textureCoordinates = make([]byte, rcvr.TriangleCount)
		}
		if rcvr.textureTriangleCount > 0 {
			rcvr.textureType = make([]byte, rcvr.textureTriangleCount)
			rcvr.texturesFaceA = make([]int16, rcvr.textureTriangleCount)
			rcvr.texturesFaceB = make([]int16, rcvr.textureTriangleCount)
			rcvr.texturesFaceC = make([]int16, rcvr.textureTriangleCount)
		}
		rcvr.VerticeCount = 0
		rcvr.TriangleCount = 0
		rcvr.textureTriangleCount = 0
		texture_face := 0
		for segmentIndex := 0; segmentIndex < length; segmentIndex++ {
			build = model_segments[segmentIndex]
			if build != nil {
				for face := 0; face < build.triangleCount; face++ {
					if renderTypeFlag && build.faceDrawType != nil {
						FaceDrawType[rcvr.TriangleCount] = build[face]
					}
					if priorityFlag {
						if build.faceRenderPriorities == nil {
							faceRenderPriorities[rcvr.TriangleCount] = build.facePriority
						} else {
							faceRenderPriorities[rcvr.TriangleCount] = build[face]
						}
					}
					if alphaFlag && build.faceAlpha != nil {
						faceAlpha[rcvr.TriangleCount] = build[face]
					}
					if tSkinFlag && build.triangleTSkin != nil {
						triangleTSkin[rcvr.TriangleCount] = build[face]
					}
					if textureFlag {
						if build.texture != nil {
							texture[rcvr.TriangleCount] = build[face]
						} else {
							texture[rcvr.TriangleCount] = -1
						}
					}
					if coordinateFlag {
						if build.textureCoordinates != nil && build[face] != -1 {
							textureCoordinates[rcvr.TriangleCount] = (build[face] + texture_face).(byte)
						} else {
							textureCoordinates[rcvr.TriangleCount] = -1
						}
					}
					triangleColours[rcvr.TriangleCount] = build[face]
					FacePointA[rcvr.TriangleCount] = rcvr.getSharedVertices(build, build[face])
					FacePointB[rcvr.TriangleCount] = getSharedVertices(build, build[face])
					FacePointC[rcvr.TriangleCount] = getSharedVertices(build, build[face])
					rcvr.TriangleCount++
				}
				for textureEdge := 0; textureEdge < build.textureTriangleCount; textureEdge++ {
					texturesFaceA[rcvr.textureTriangleCount] = getSharedVertices(build, build[textureEdge]).(int16)
					texturesFaceB[rcvr.textureTriangleCount] = getSharedVertices(build, build[textureEdge]).(int16)
					texturesFaceC[rcvr.textureTriangleCount] = getSharedVertices(build, build[textureEdge]).(int16)
					rcvr.textureTriangleCount++
				}
				texture_face += build.textureTriangleCount
			}
		}
	} else if catch_Exception(exception) {
		exception.printStackTrace()
	}
	return
}
func NewModel3(colorFlag bool, alphaFlag bool, animated bool, textureFlag bool, model *Model) (rcvr *Model) {
	rcvr = &Model{}
	rcvr.fitsOnSingleTile = false
	rcvr.VerticeCount = model.verticeCount
	rcvr.TriangleCount = model.triangleCount
	rcvr.textureTriangleCount = model.textureTriangleCount
	if animated {
		rcvr.verticesParticle = model.verticesParticle
		rcvr.VertexX = model.vertexX
		rcvr.VertexY = model.vertexY
		rcvr.VertexZ = model.vertexZ
	} else {
		rcvr.verticesParticle = make([]int, rcvr.VerticeCount)
		rcvr.VertexX = make([]int, rcvr.VerticeCount)
		rcvr.VertexY = make([]int, rcvr.VerticeCount)
		rcvr.VertexZ = make([]int, rcvr.VerticeCount)
		for index := 0; index < rcvr.VerticeCount; index++ {
			verticesParticle[index] = model[index]
			VertexX[index] = model[index]
			VertexY[index] = model[index]
			VertexZ[index] = model[index]
		}
	}
	if colorFlag {
		rcvr.triangleColours = model.triangleColours
	} else {
		rcvr.triangleColours = make([]int16, rcvr.TriangleCount)
		for index := 0; index < rcvr.TriangleCount; index++ {
			triangleColours[index] = model[index]
		}
	}
	if !textureFlag && model.texture != nil {
		rcvr.texture = make([]int16, rcvr.TriangleCount)
		for face := 0; face < rcvr.TriangleCount; face++ {
			texture[face] = model[face]
		}
	} else {
		rcvr.texture = model.texture
	}
	if alphaFlag {
		rcvr.faceAlpha = model.faceAlpha
	} else {
		rcvr.faceAlpha = make([]int, rcvr.TriangleCount)
		if model.faceAlpha == nil {
			for index := 0; index < rcvr.TriangleCount; index++ {
				faceAlpha[index] = 0
			}
		} else {
			for index := 0; index < rcvr.TriangleCount; index++ {
				faceAlpha[index] = model[index]
			}
		}
	}
	rcvr.vertexVSkin = model.vertexVSkin
	rcvr.triangleTSkin = model.triangleTSkin
	rcvr.FaceDrawType = model.faceDrawType
	rcvr.FacePointA = model.facePointA
	rcvr.FacePointB = model.facePointB
	rcvr.FacePointC = model.facePointC
	rcvr.faceRenderPriorities = model.faceRenderPriorities
	rcvr.facePriority = model.facePriority
	rcvr.texturesFaceA = model.texturesFaceA
	rcvr.texturesFaceB = model.texturesFaceB
	rcvr.texturesFaceC = model.texturesFaceC
	rcvr.textureCoordinates = model.textureCoordinates
	rcvr.textureType = model.textureType
	return
}
func NewModel4(modelId int) (rcvr *Model) {
	rcvr = &Model{}
	modelData := <<unimp_obj.nm_*parser.GoArrayReference>>
	if modelData[len(modelData)-1] == -1 && modelData[len(modelData)-2] == -1 {
		rcvr.decodeNew(modelData, modelId)
	} else {
		rcvr.DecodeOld(modelData, modelId)
	}
	return
}
func (rcvr *Model) calculateDistances() {
	<<super>> = 0
	rcvr.MaxVertexDistanceXZPlane = 0
	rcvr.MaximumYVertex = 0
	for i := 0; i < rcvr.VerticeCount; i++ {
		x := VertexX[i]
		y := VertexY[i]
		z := VertexZ[i]
		if -y > <<super>> {
			<<super>> = -y
		}
		if y > rcvr.MaximumYVertex {
			rcvr.MaximumYVertex = y
		}
		sqDistance := x*x + z*z
		if sqDistance > rcvr.MaxVertexDistanceXZPlane {
			rcvr.MaxVertexDistanceXZPlane = sqDistance
		}
	}
	rcvr.MaxVertexDistanceXZPlane = (Math.sqrt(rcvr.MaxVertexDistanceXZPlane) + 0.98999999999999999D).(int)
	rcvr.diagonal3DAboveOrigin = (Math.sqrt(rcvr.MaxVertexDistanceXZPlane*rcvr.MaxVertexDistanceXZPlane+<<super>>*<<super>>) + 0.98999999999999999D).(int)
	rcvr.maxRenderDepth = rcvr.diagonal3DAboveOrigin + (Math.sqrt(rcvr.MaxVertexDistanceXZPlane*rcvr.MaxVertexDistanceXZPlane+rcvr.MaximumYVertex*rcvr.MaximumYVertex) + 0.98999999999999999D).(int)
}
func (rcvr *Model) calculateVertexData() {
	<<super>> = 0
	rcvr.MaxVertexDistanceXZPlane = 0
	rcvr.MaximumYVertex = 0
	rcvr.MinimumXVertex = 999999
	rcvr.MaximumXVertex = -999999
	rcvr.MaximumZVertex = -99999
	rcvr.MinimumZVertex = 99999
	for idx := 0; idx < rcvr.VerticeCount; idx++ {
		xVertex := VertexX[idx]
		yVertex := VertexY[idx]
		zVertex := VertexZ[idx]
		if xVertex < rcvr.MinimumXVertex {
			rcvr.MinimumXVertex = xVertex
		}
		if xVertex > rcvr.MaximumXVertex {
			rcvr.MaximumXVertex = xVertex
		}
		if zVertex < rcvr.MinimumZVertex {
			rcvr.MinimumZVertex = zVertex
		}
		if zVertex > rcvr.MaximumZVertex {
			rcvr.MaximumZVertex = zVertex
		}
		if -yVertex > <<super>> {
			<<super>> = -yVertex
		}
		if yVertex > rcvr.MaximumYVertex {
			rcvr.MaximumYVertex = yVertex
		}
		vertexDistanceXZPlane := xVertex*xVertex + zVertex*zVertex
		if vertexDistanceXZPlane > rcvr.MaxVertexDistanceXZPlane {
			rcvr.MaxVertexDistanceXZPlane = vertexDistanceXZPlane
		}
	}
	rcvr.MaxVertexDistanceXZPlane = Math.sqrt(rcvr.MaxVertexDistanceXZPlane).(int)
	rcvr.diagonal3DAboveOrigin = Math.sqrt(rcvr.MaxVertexDistanceXZPlane*rcvr.MaxVertexDistanceXZPlane + <<super>>*<<super>>).(int)
	rcvr.maxRenderDepth = rcvr.diagonal3DAboveOrigin + Math.sqrt(rcvr.MaxVertexDistanceXZPlane*rcvr.MaxVertexDistanceXZPlane+rcvr.MaximumYVertex*rcvr.MaximumYVertex).(int)
}
func (rcvr *Model) ComputeSphericalBounds() {
	<<super>> = 0
	rcvr.MaximumYVertex = 0
	for index := 0; index < rcvr.VerticeCount; index++ {
		y := VertexY[index]
		if -y > <<super>> {
			<<super>> = -y
		}
		if y > rcvr.MaximumYVertex {
			rcvr.MaximumYVertex = y
		}
	}
	rcvr.diagonal3DAboveOrigin = (Math.sqrt(rcvr.MaxVertexDistanceXZPlane*rcvr.MaxVertexDistanceXZPlane+<<super>>*<<super>>) + 0.98999999999999999D).(int)
	rcvr.maxRenderDepth = rcvr.diagonal3DAboveOrigin + (Math.sqrt(rcvr.MaxVertexDistanceXZPlane*rcvr.MaxVertexDistanceXZPlane+rcvr.MaximumYVertex*rcvr.MaximumYVertex) + 0.98999999999999999D).(int)
}
func DecodeHeader(data []byte, j int) {
	if try() {
		if data == nil {
			modelHeader := modelHeaderCache[j] = NewModelHeader()
			modelHeader.modelVerticeCount = 0
			modelHeader.modelTriangleCount = 0
			modelHeader.modelTextureTriangleCount = 0
			return
		}
		stream := NewBuffer(data)
		stream.currentPosition = len(data) - 18
		modelHeader := modelHeaderCache[j] = NewModelHeader()
		modelHeader.modelData = data
		modelHeader.modelVerticeCount = stream.readUShort()
		modelHeader.modelTriangleCount = stream.readUShort()
		modelHeader.modelTextureTriangleCount = stream.readUnsignedByte()
		k := stream.readUnsignedByte()
		l := stream.readUnsignedByte()
		i1 := stream.readUnsignedByte()
		j1 := stream.readUnsignedByte()
		k1 := stream.readUnsignedByte()
		l1 := stream.readUShort()
		i2 := stream.readUShort()
		j2 := stream.readUShort()
		k2 := stream.readUShort()
		l2 := 0
		modelHeader.vertexModOffset = l2
		l2 += modelHeader.modelVerticeCount
		modelHeader.triMeshLinkOffset = l2
		l2 += modelHeader.modelTriangleCount
		modelHeader.facePriorityBasePos = l2
		if l == 255 {
			l2 += modelHeader.modelTriangleCount
		} else {
			modelHeader.facePriorityBasePos = -l - 1
		}
		modelHeader.tskinBasepos = l2
		if j1 == 1 {
			l2 += modelHeader.modelTriangleCount
		} else {
			modelHeader.tskinBasepos = -1
		}
		modelHeader.drawTypeBasePos = l2
		if k == 1 {
			l2 += modelHeader.modelTriangleCount
		} else {
			modelHeader.drawTypeBasePos = -1
		}
		modelHeader.vskinBasePos = l2
		if k1 == 1 {
			l2 += modelHeader.modelVerticeCount
		} else {
			modelHeader.vskinBasePos = -1
		}
		modelHeader.alphaBasepos = l2
		if i1 == 1 {
			l2 += modelHeader.modelTriangleCount
		} else {
			modelHeader.alphaBasepos = -1
		}
		modelHeader.triVPointOffset = l2
		l2 += k2
		modelHeader.triColourOffset = l2
		l2 += modelHeader.modelTriangleCount * 2
		modelHeader.textureInfoBasePos = l2
		l2 += modelHeader.modelTextureTriangleCount * 6
		modelHeader.vertexXOffset = l2
		l2 += l1
		modelHeader.vertexYOffset = l2
		l2 += i2
		modelHeader.vertexZOffset = l2
		l2 += j2
	} else if catch_Exception(_ex) {
		_ex.printStackTrace()
	}
}
func (rcvr *Model) decodeNew(data []byte, modelId int) {
	first := NewBuffer(data)
	second := NewBuffer(data)
	third := NewBuffer(data)
	fourth := NewBuffer(data)
	fifth := NewBuffer(data)
	sixth := NewBuffer(data)
	seventh := NewBuffer(data)
	first.currentPosition = len(data) - 23
	rcvr.VerticeCount = first.readUShort()
	rcvr.TriangleCount = first.readUShort()
	rcvr.textureTriangleCount = first.readUnsignedByte()
	renderTypeOpcode := first.readUnsignedByte()
	priorityOpcode := first.readUnsignedByte()
	alphaOpcode := first.readUnsignedByte()
	tSkinOpcode := first.readUnsignedByte()
	textureOpcode := first.readUnsignedByte()
	vSkinOpcode := first.readUnsignedByte()
	vertexX := first.readUShort()
	vertexY := first.readUShort()
	vertexZ := first.readUShort()
	vertexPoints := first.readUShort()
	textureIndices := first.readUShort()
	textureIdSimple := 0
	textureIdComplex := 0
	textureIdCube := 0
	var face int
	rcvr.triangleColours = make([]int16, rcvr.TriangleCount)
	if rcvr.textureTriangleCount > 0 {
		rcvr.textureType = make([]byte, rcvr.textureTriangleCount)
		first.currentPosition = 0
		for face = 0; face < rcvr.textureTriangleCount; ++face {
			opcode := textureType[face] = first.readSignedByte()
			if opcode == 0 {
				textureIdSimple++
			}
			if opcode >= 1 && opcode <= 3 {
				textureIdComplex++
			}
			if opcode == 2 {
				textureIdCube++
			}
		}
	}
	var position int
	position = rcvr.textureTriangleCount
	vertexOffset := position
	position += rcvr.VerticeCount
	renderTypeOffset := position
	if renderTypeOpcode == 1 {
		position += rcvr.TriangleCount
	}
	faceOffset := position
	position += rcvr.TriangleCount
	facePriorityOffset := position
	if priorityOpcode == 255 {
		position += rcvr.TriangleCount
	}
	tSkinOffset := position
	if tSkinOpcode == 1 {
		position += rcvr.TriangleCount
	}
	vSkinOffset := position
	if vSkinOpcode == 1 {
		position += rcvr.VerticeCount
	}
	alphaOffset := position
	if alphaOpcode == 1 {
		position += rcvr.TriangleCount
	}
	pointsOffset := position
	position += vertexPoints
	textureId := position
	if textureOpcode == 1 {
		position += rcvr.TriangleCount * 2
	}
	textureCoordinateOffset := position
	position += textureIndices
	colorOffset := position
	position += rcvr.TriangleCount * 2
	vertexXOffset := position
	position += vertexX
	vertexYOffset := position
	position += vertexY
	vertexZOffset := position
	position += vertexZ
	simpleTextureoffset := position
	position += textureIdSimple * 6
	complexTextureOffset := position
	position += textureIdComplex * 6
	textureScalOffset := position
	position += textureIdComplex * 6
	textureRotationOffset := position
	position += textureIdComplex * 2
	textureDirectionOffset := position
	position += textureIdComplex
	textureTranslateOffset := position
	position += textureIdComplex*2 + textureIdCube*2
	rcvr.verticesParticle = make([]int, rcvr.VerticeCount)
	rcvr.vertexX = make([]int, rcvr.VerticeCount)
	rcvr.vertexY = make([]int, rcvr.VerticeCount)
	rcvr.vertexZ = make([]int, rcvr.VerticeCount)
	rcvr.FacePointA = make([]int, rcvr.TriangleCount)
	rcvr.FacePointB = make([]int, rcvr.TriangleCount)
	rcvr.FacePointC = make([]int, rcvr.TriangleCount)
	if vSkinOpcode == 1 {
		rcvr.vertexVSkin = make([]int, rcvr.VerticeCount)
	}
	if renderTypeOpcode == 1 {
		rcvr.FaceDrawType = make([]int, rcvr.TriangleCount)
	}
	if priorityOpcode == 255 {
		rcvr.faceRenderPriorities = make([]byte, rcvr.TriangleCount)
	} else {
		rcvr.facePriority = priorityOpcode.(byte)
	}
	if alphaOpcode == 1 {
		rcvr.faceAlpha = make([]int, rcvr.TriangleCount)
	}
	if tSkinOpcode == 1 {
		rcvr.triangleTSkin = make([]int, rcvr.TriangleCount)
	}
	if textureOpcode == 1 {
		rcvr.texture = make([]int16, rcvr.TriangleCount)
	}
	if textureOpcode == 1 && rcvr.textureTriangleCount > 0 {
		rcvr.textureCoordinates = make([]byte, rcvr.TriangleCount)
	}
	if rcvr.textureTriangleCount > 0 {
		rcvr.texturesFaceA = make([]int16, rcvr.textureTriangleCount)
		rcvr.texturesFaceB = make([]int16, rcvr.textureTriangleCount)
		rcvr.texturesFaceC = make([]int16, rcvr.textureTriangleCount)
	}
	first.currentPosition = vertexOffset
	second.currentPosition = vertexXOffset
	third.currentPosition = vertexYOffset
	fourth.currentPosition = vertexZOffset
	fifth.currentPosition = vSkinOffset
	startX := 0
	startY := 0
	startZ := 0
	for point := 0; point < rcvr.VerticeCount; point++ {
		positionMask := first.readUnsignedByte()
		x := 0
		if positionMask&1 != 0 {
			x = second.readSmart()
		}
		y := 0
		if positionMask&2 != 0 {
			y = third.readSmart()
		}
		z := 0
		if positionMask&4 != 0 {
			z = fourth.readSmart()
		}
		rcvr.vertexX[point] = startX + x
		rcvr.vertexY[point] = startY + y
		rcvr.vertexZ[point] = startZ + z
		startX = rcvr.vertexX[point]
		startY = rcvr.vertexY[point]
		startZ = rcvr.vertexZ[point]
		if rcvr.vertexVSkin != nil {
			vertexVSkin[point] = fifth.readUnsignedByte()
		}
	}
	first.currentPosition = colorOffset
	second.currentPosition = renderTypeOffset
	third.currentPosition = facePriorityOffset
	fourth.currentPosition = alphaOffset
	fifth.currentPosition = tSkinOffset
	sixth.currentPosition = textureId
	seventh.currentPosition = textureCoordinateOffset
	for face = 0; face < rcvr.TriangleCount; ++face {
		triangleColours[face] = first.readUShort().(int16)
		if renderTypeOpcode == 1 {
			FaceDrawType[face] = second.readSignedByte()
		}
		if priorityOpcode == 255 {
			faceRenderPriorities[face] = third.readSignedByte()
		}
		if alphaOpcode == 1 {
			faceAlpha[face] = fourth.readSignedByte()
			if faceAlpha[face] < 0 {
				faceAlpha[face] = 256 + faceAlpha[face]
			}
		}
		if tSkinOpcode == 1 {
			triangleTSkin[face] = fifth.readUnsignedByte()
		}
		if textureOpcode == 1 {
			texture[face] = (sixth.readUShort() - 1).(int16)
			if texture[face] >= 0 {
				if rcvr.FaceDrawType != nil {
					if FaceDrawType[face] < 2 && triangleColours[face] != 127 && triangleColours[face] != -27075 {
						texture[face] = -1
					}
				}
			}
			if texture[face] != -1 {
				triangleColours[face] = 127
			}
		}
		if rcvr.textureCoordinates != nil && texture[face] != -1 {
			textureCoordinates[face] = (seventh.readUnsignedByte() - 1).(byte)
		}
	}
	first.currentPosition = pointsOffset
	second.currentPosition = faceOffset
	coordinateA := 0
	coordinateB := 0
	coordinateC := 0
	offset := 0
	for face = 0; face < rcvr.TriangleCount; ++face {
		opcode := second.readUnsignedByte()
		if opcode == 1 {
			coordinateA = first.readSmart() + offset
			offset = coordinateA
			coordinateB = first.readSmart() + offset
			offset = coordinateB
			coordinateC = first.readSmart() + offset
			offset = coordinateC
			FacePointA[face] = coordinateA
			FacePointB[face] = coordinateB
			FacePointC[face] = coordinateC
		}
		if opcode == 2 {
			coordinateB = coordinateC
			coordinateC = first.readSmart() + offset
			offset = coordinateC
			FacePointA[face] = coordinateA
			FacePointB[face] = coordinateB
			FacePointC[face] = coordinateC
		}
		if opcode == 3 {
			coordinateA = coordinateC
			coordinateC = first.readSmart() + offset
			offset = coordinateC
			FacePointA[face] = coordinateA
			FacePointB[face] = coordinateB
			FacePointC[face] = coordinateC
		}
		if opcode == 4 {
			tempCoordinateA := coordinateA
			coordinateA = coordinateB
			coordinateB = tempCoordinateA
			coordinateC = first.readSmart() + offset
			offset = coordinateC
			FacePointA[face] = coordinateA
			FacePointB[face] = coordinateB
			FacePointC[face] = coordinateC
		}
	}
	first.currentPosition = simpleTextureoffset
	second.currentPosition = complexTextureOffset
	third.currentPosition = textureScalOffset
	fourth.currentPosition = textureRotationOffset
	fifth.currentPosition = textureDirectionOffset
	sixth.currentPosition = textureTranslateOffset
	for face = 0; face < rcvr.textureTriangleCount; ++face {
		opcode := textureType[face] & 0xff
		if opcode == 0 {
			texturesFaceA[face] = first.readUShort().(int16)
			texturesFaceB[face] = first.readUShort().(int16)
			texturesFaceC[face] = first.readUShort().(int16)
		}
		if opcode == 1 {
			texturesFaceA[face] = second.readUShort().(int16)
			texturesFaceB[face] = second.readUShort().(int16)
			texturesFaceC[face] = second.readUShort().(int16)
		}
		if opcode == 2 {
			texturesFaceA[face] = second.readUShort().(int16)
			texturesFaceB[face] = second.readUShort().(int16)
			texturesFaceC[face] = second.readUShort().(int16)
		}
		if opcode == 3 {
			texturesFaceA[face] = second.readUShort().(int16)
			texturesFaceB[face] = second.readUShort().(int16)
			texturesFaceC[face] = second.readUShort().(int16)
		}
	}
	first.currentPosition = vertexOffset
	face = first.readUnsignedByte()
}
func (rcvr *Model) DecodeOld(data []byte, modelId int) {
	hasFaceType := false
	hasTexture_Type := false
	first := NewBuffer(data)
	second := NewBuffer(data)
	third := NewBuffer(data)
	fourth := NewBuffer(data)
	fifth := NewBuffer(data)
	first.currentPosition = len(data) - 18
	rcvr.VerticeCount = first.readUShort()
	rcvr.TriangleCount = first.readUShort()
	rcvr.textureTriangleCount = first.readUnsignedByte()
	renderTypeOpcode := first.readUnsignedByte()
	priorityOpcode := first.readUnsignedByte()
	alphaOpcode := first.readUnsignedByte()
	tSkinOpcode := first.readUnsignedByte()
	vSkinOpcode := first.readUnsignedByte()
	vertexX := first.readUShort()
	vertexY := first.readUShort()
	vertexZ := first.readUShort()
	vertexPoints := first.readUShort()
	position := 0
	vertexFlagOffset := position
	position += rcvr.VerticeCount
	faceCompressTypeOffset := position
	position += rcvr.TriangleCount
	facePriorityOffset := position
	if priorityOpcode == 255 {
		position += rcvr.TriangleCount
	}
	tSkinOffset := position
	if tSkinOpcode == 1 {
		position += rcvr.TriangleCount
	}
	renderTypeOffset := position
	if renderTypeOpcode == 1 {
		position += rcvr.TriangleCount
	}
	vSkinOffset := position
	if vSkinOpcode == 1 {
		position += rcvr.VerticeCount
	}
	alphaOffset := position
	if alphaOpcode == 1 {
		position += rcvr.TriangleCount
	}
	pointsOffset := position
	position += vertexPoints
	colorOffset := position
	position += rcvr.TriangleCount * 2
	textureOffset := position
	position += rcvr.textureTriangleCount * 6
	vertexXOffset := position
	position += vertexX
	vertexYOffset := position
	position += vertexY
	vertexZOffset := position
	position += vertexZ
	rcvr.verticesParticle = make([]int, rcvr.VerticeCount)
	rcvr.vertexX = make([]int, rcvr.VerticeCount)
	rcvr.vertexY = make([]int, rcvr.VerticeCount)
	rcvr.vertexZ = make([]int, rcvr.VerticeCount)
	rcvr.FacePointA = make([]int, rcvr.TriangleCount)
	rcvr.FacePointB = make([]int, rcvr.TriangleCount)
	rcvr.FacePointC = make([]int, rcvr.TriangleCount)
	if rcvr.textureTriangleCount > 0 {
		rcvr.textureType = make([]byte, rcvr.textureTriangleCount)
		rcvr.texturesFaceA = make([]int16, rcvr.textureTriangleCount)
		rcvr.texturesFaceB = make([]int16, rcvr.textureTriangleCount)
		rcvr.texturesFaceC = make([]int16, rcvr.textureTriangleCount)
	}
	if vSkinOpcode == 1 {
		rcvr.vertexVSkin = make([]int, rcvr.VerticeCount)
	}
	if renderTypeOpcode == 1 {
		rcvr.FaceDrawType = make([]int, rcvr.TriangleCount)
		rcvr.textureCoordinates = make([]byte, rcvr.TriangleCount)
		rcvr.texture = make([]int16, rcvr.TriangleCount)
	}
	if priorityOpcode == 255 {
		rcvr.faceRenderPriorities = make([]byte, rcvr.TriangleCount)
	} else {
		rcvr.facePriority = priorityOpcode.(byte)
	}
	if alphaOpcode == 1 {
		rcvr.faceAlpha = make([]int, rcvr.TriangleCount)
	}
	if tSkinOpcode == 1 {
		rcvr.triangleTSkin = make([]int, rcvr.TriangleCount)
	}
	rcvr.triangleColours = make([]int16, rcvr.TriangleCount)
	first.currentPosition = vertexFlagOffset
	second.currentPosition = vertexXOffset
	third.currentPosition = vertexYOffset
	fourth.currentPosition = vertexZOffset
	fifth.currentPosition = vSkinOffset
	startX := 0
	startY := 0
	startZ := 0
	for point := 0; point < rcvr.VerticeCount; point++ {
		positionMask := first.readUnsignedByte()
		x := 0
		if positionMask&0x1 != 0 {
			x = second.readSmart()
		}
		y := 0
		if positionMask&0x2 != 0 {
			y = third.readSmart()
		}
		z := 0
		if positionMask&0x4 != 0 {
			z = fourth.readSmart()
		}
		rcvr.vertexX[point] = startX + x
		rcvr.vertexY[point] = startY + y
		rcvr.vertexZ[point] = startZ + z
		startX = rcvr.vertexX[point]
		startY = rcvr.vertexY[point]
		startZ = rcvr.vertexZ[point]
		if vSkinOpcode == 1 {
			vertexVSkin[point] = fifth.readUnsignedByte()
		}
	}
	first.currentPosition = colorOffset
	second.currentPosition = renderTypeOffset
	third.currentPosition = facePriorityOffset
	fourth.currentPosition = alphaOffset
	fifth.currentPosition = tSkinOffset
	for face := 0; face < rcvr.TriangleCount; face++ {
		triangleColours[face] = first.readUShort().(int16)
		if renderTypeOpcode == 1 {
			flag := second.readUnsignedByte()
			if flag&0x1 == 1 {
				FaceDrawType[face] = 1
				hasFaceType = true
			} else {
				FaceDrawType[face] = 0
			}
			if flag&0x2 != 0 {
				textureCoordinates[face] = (uint32(flag) >> 2).(byte)
				texture[face] = triangleColours[face]
				triangleColours[face] = 127
				if texture[face] != -1 {
					hasTexture_Type = true
				}
			} else {
				textureCoordinates[face] = -1
				texture[face] = -1
			}
		}
		if priorityOpcode == 255 {
			faceRenderPriorities[face] = third.readSignedByte()
		}
		if alphaOpcode == 1 {
			faceAlpha[face] = fourth.readSignedByte()
			if faceAlpha[face] < 0 {
				faceAlpha[face] = 256 + faceAlpha[face]
			}
		}
		if tSkinOpcode == 1 {
			triangleTSkin[face] = fifth.readUnsignedByte()
		}
	}
	first.currentPosition = pointsOffset
	second.currentPosition = faceCompressTypeOffset
	coordinateA := 0
	coordinateB := 0
	coordinateC := 0
	offset := 0
	var coordinate int
	for face := 0; face < rcvr.TriangleCount; face++ {
		opcode := second.readUnsignedByte()
		if opcode == 1 {
			coordinateA = first.readSmart() + offset
			offset = coordinateA
			coordinateB = first.readSmart() + offset
			offset = coordinateB
			coordinateC = first.readSmart() + offset
			offset = coordinateC
			FacePointA[face] = coordinateA
			FacePointB[face] = coordinateB
			FacePointC[face] = coordinateC
		}
		if opcode == 2 {
			coordinateB = coordinateC
			coordinateC = first.readSmart() + offset
			offset = coordinateC
			FacePointA[face] = coordinateA
			FacePointB[face] = coordinateB
			FacePointC[face] = coordinateC
		}
		if opcode == 3 {
			coordinateA = coordinateC
			coordinateC = first.readSmart() + offset
			offset = coordinateC
			FacePointA[face] = coordinateA
			FacePointB[face] = coordinateB
			FacePointC[face] = coordinateC
		}
		if opcode == 4 {
			coordinate = coordinateA
			coordinateA = coordinateB
			coordinateB = coordinate
			coordinateC = first.readSmart() + offset
			offset = coordinateC
			FacePointA[face] = coordinateA
			FacePointB[face] = coordinateB
			FacePointC[face] = coordinateC
		}
	}
	first.currentPosition = textureOffset
	for face := 0; face < rcvr.textureTriangleCount; face++ {
		textureType[face] = 0
		texturesFaceA[face] = first.readUShort().(int16)
		texturesFaceB[face] = first.readUShort().(int16)
		texturesFaceC[face] = first.readUShort().(int16)
	}
	if rcvr.textureCoordinates != nil {
		textured := false
		for face := 0; face < rcvr.TriangleCount; face++ {
			coordinate = textureCoordinates[face] & 0xff
			if coordinate != 255 {
				if texturesFaceA[coordinate]&0xffff == FacePointA[face] && texturesFaceB[coordinate]&0xffff == FacePointB[face] && texturesFaceC[coordinate]&0xffff == FacePointC[face] {
					textureCoordinates[face] = -1
				} else {
					textured = true
				}
			}
		}
		if !textured {
			rcvr.textureCoordinates = nil
		}
	}
	if !hasTexture_Type {
		rcvr.texture = nil
	}
	if !hasFaceType {
		rcvr.FaceDrawType = nil
	}
}
func (rcvr *Model) FlatLighting(intensity int, distributionFactor int, lightX int, lightY int, lightZ int) {
	for triangle := 0; triangle < rcvr.TriangleCount; triangle++ {
		a := FacePointA[triangle]
		b := FacePointB[triangle]
		c := FacePointC[triangle]
		var textureId int16
		if rcvr.texture == nil {
			textureId = -1
		} else {
			textureId = texture[triangle]
		}
		if rcvr.FaceDrawType == nil {
			var type int
			if textureId != -1 {
				type = 2
			} else {
				type = 1
			}
			hsl := triangleColours[triangle]
			vertexNormal := <<super>>[a]
			lightItensity := intensity + (lightX*vertexNormal.normalX+lightY*vertexNormal.normalY+lightZ*vertexNormal.normalZ)/(distributionFactor*vertexNormal.magnitude)
			faceHslA[triangle] = Model.Light(hsl, lightItensity, type)
			vertexNormal = <<super>>[b]
			lightItensity = intensity + (lightX*vertexNormal.normalX+lightY*vertexNormal.normalY+lightZ*vertexNormal.normalZ)/(distributionFactor*vertexNormal.magnitude)
			faceHslB[triangle] = Model.Light(hsl, lightItensity, type)
			vertexNormal = <<super>>[c]
			lightItensity = intensity + (lightX*vertexNormal.normalX+lightY*vertexNormal.normalY+lightZ*vertexNormal.normalZ)/(distributionFactor*vertexNormal.magnitude)
			faceHslC[triangle] = Model.Light(hsl, lightItensity, type)
		} else if FaceDrawType[triangle]&1 == 0 {
			hsl := triangleColours[triangle]
			type := FaceDrawType[triangle]
			if textureId != -1 {
				type = 2
			}
			vertexNormal := <<super>>[a]
			lightItensity := intensity + (lightX*vertexNormal.normalX+lightY*vertexNormal.normalY+lightZ*vertexNormal.normalZ)/(distributionFactor*vertexNormal.magnitude)
			faceHslA[triangle] = Model.Light(hsl, lightItensity, type)
			vertexNormal = <<super>>[b]
			lightItensity = intensity + (lightX*vertexNormal.normalX+lightY*vertexNormal.normalY+lightZ*vertexNormal.normalZ)/(distributionFactor*vertexNormal.magnitude)
			faceHslB[triangle] = Model.Light(hsl, lightItensity, type)
			vertexNormal = <<super>>[c]
			lightItensity = intensity + (lightX*vertexNormal.normalX+lightY*vertexNormal.normalY+lightZ*vertexNormal.normalZ)/(distributionFactor*vertexNormal.magnitude)
			faceHslC[triangle] = Model.Light(hsl, lightItensity, type)
		}
	}
	<<super>> = nil
	rcvr.AlsoVertexNormals = nil
	rcvr.vertexVSkin = nil
	rcvr.triangleTSkin = nil
	rcvr.triangleColours = nil
}
func Get(file int) (*Model) {
	if modelHeaderCache == nil {
		return nil
	}
	modelHeader := modelHeaderCache[file]
	if modelHeader == nil {
		Model.DecodeHeader(resourceProvider.getModel(file), file)
		return NewModel4(file)
	} else {
		return NewModel4(file)
	}
}
func (rcvr *Model) getSharedVertices(model *Model, point int) (int) {
	sharedVertex := -1
	particlePoint := model[point]
	x := model[point]
	y := model[point]
	z := model[point]
	for index := 0; index < rcvr.VerticeCount; index++ {
		if x != VertexX[index] || y != VertexY[index] || z != VertexZ[index] {
			continue
		}
		sharedVertex = index
		break
	}
	if sharedVertex == -1 {
		verticesParticle[rcvr.VerticeCount] = particlePoint
		VertexX[rcvr.VerticeCount] = x
		VertexY[rcvr.VerticeCount] = y
		VertexZ[rcvr.VerticeCount] = z
		if model.vertexVSkin != nil {
			vertexVSkin[rcvr.VerticeCount] = model[point]
		}
		sharedVertex = ++rcvr.VerticeCount
	}
	return sharedVertex
}
func init() {
	Sine = Rasterizer3D.sine
	Cosine = Rasterizer3D.cosine
	modelIntArray3 = Rasterizer3D.hslToRgb
	modelIntArray4 = Rasterizer3D.DEPTH
}
func Initialize(modelAmount int, resourceProviderInstance *MapDefinition) {
	modelHeaderCache = make([]*ModelHeader, modelAmount)
	resourceProvider = resourceProviderInstance
}
func (rcvr *Model) Invert() {
	for index := 0; index < rcvr.VerticeCount; index++ {
		VertexZ[index] = -VertexZ[index]
	}
	for face := 0; face < rcvr.TriangleCount; face++ {
		triA := FacePointA[face]
		FacePointA[face] = FacePointC[face]
		FacePointC[face] = triA
	}
}
func Light(hsl int, light int, type int) (int) {
	if hsl == 65535 {
		return 0
	}
	if type&2 == 2 {
		return light(light)
	}
	return light(hsl, light)
}
func Light2(light int) (int) {
	if light < 0 {
		light = 0
	} else if light > 127 {
		light = 127
	}
	light = 127 - light
	return light
}
func Light3(hsl int, light int) (int) {
	light = uint32(light*(hsl&0x7f)) >> 7
	if light < 2 {
		light = 2
	} else if light > 126 {
		light = 126
	}
	return hsl&0xff80 + light
}
func (rcvr *Model) Light4(i int, j int, k int, l int, i1 int, lightModelNotSure bool) {
	j1, ok := Math.sqrt(k*k + l*l + i1*i1).(int)
	if !ok {
		panic("XXX Cast fail for *parser.GoCastType")
	}
	k1 := uint32(j*j1) >> 8
	if rcvr.faceHslA == nil {
		rcvr.faceHslA = make([]int, rcvr.TriangleCount)
		rcvr.faceHslB = make([]int, rcvr.TriangleCount)
		rcvr.faceHslC = make([]int, rcvr.TriangleCount)
	}
	if <<super>> == nil {
		<<super>> = make([]*VertexNormal, rcvr.VerticeCount)
		for l1 := 0; l1 < rcvr.VerticeCount; l1++ {
			<<super>>[l1] = NewVertexNormal()
		}
	}
	for i2 := 0; i2 < rcvr.TriangleCount; i2++ {
		j2 := FacePointA[i2]
		l2 := FacePointB[i2]
		i3 := FacePointC[i2]
		j3 := VertexX[l2] - VertexX[j2]
		k3 := VertexY[l2] - VertexY[j2]
		l3 := VertexZ[l2] - VertexZ[j2]
		i4 := VertexX[i3] - VertexX[j2]
		j4 := VertexY[i3] - VertexY[j2]
		k4 := VertexZ[i3] - VertexZ[j2]
		l4 := k3*k4 - j4*l3
		i5 := l3*i4 - k4*j3
		var j5 int
		for j5 = j3*j4-i4*k3; l4 > 8192 || i5 > 8192 || j5 > 8192 || l4 < -8192 || i5 < -8192 || j5 < -8192; j5 = uint32(j5)>>1 {
			l4 = uint32(l4) >> 1
			i5 = uint32(i5) >> 1
		}
		k5, ok := Math.sqrt(l4*l4 + i5*i5 + j5*j5).(int)
		if !ok {
			panic("XXX Cast fail for *parser.GoCastType")
		}
		if k5 <= 0 {
			k5 = 1
		}
		l4 = l4 * 256 / k5
		i5 = i5 * 256 / k5
		j5 = j5 * 256 / k5
		var texture_id int16
		var type int
		if rcvr.FaceDrawType != nil {
			type = FaceDrawType[i2]
		} else {
			type = 0
		}
		if rcvr.texture == nil {
			texture_id = -1
		} else {
			texture_id = texture[i2]
		}
		if rcvr.FaceDrawType == nil || FaceDrawType[i2]&1 == 0 {
			vertexNormal := <<super>>[j2]
			vertexNormal.normalX += l4
			vertexNormal.normalY += i5
			vertexNormal.normalZ += j5
			vertexNormal.magnitude++
			vertexNormal = <<super>>[l2]
			vertexNormal.normalX += l4
			vertexNormal.normalY += i5
			vertexNormal.normalZ += j5
			vertexNormal.magnitude++
			vertexNormal = <<super>>[i3]
			vertexNormal.normalX += l4
			vertexNormal.normalY += i5
			vertexNormal.normalZ += j5
			vertexNormal.magnitude++
		} else {
			if texture_id != -1 {
				type = 2
			}
			l5 := i + (k*l4+l*i5+i1*j5)/(k1+k1/2)
			faceHslA[i2] = light(triangleColours[i2], l5, type)
		}
	}
	if lightModelNotSure {
		rcvr.FlatLighting(i, k1, k, l, i1)
	} else {
		rcvr.AlsoVertexNormals = make([]*VertexNormal, rcvr.VerticeCount)
		for k2 := 0; k2 < rcvr.VerticeCount; k2++ {
			vertexNormal := <<super>>[k2]
			vertexMerge := AlsoVertexNormals[k2] = NewVertexNormal()
			vertexMerge.normalX = vertexNormal.normalX
			vertexMerge.normalY = vertexNormal.normalY
			vertexMerge.normalZ = vertexNormal.normalZ
			vertexMerge.magnitude = vertexNormal.magnitude
		}
	}
	if lightModelNotSure {
		rcvr.calculateDistances()
	} else {
		rcvr.calculateVertexData()
	}
}
func (rcvr *Model) method483(flag bool, flag1 bool, i int) {
	for j := 0; j < rcvr.maxRenderDepth; j++ {
		depthListIndices[j] = 0
	}
	for k := 0; k < rcvr.TriangleCount; k++ {
		if rcvr.FaceDrawType == nil || FaceDrawType[k] != -1 {
			l := FacePointA[k]
			k1 := FacePointB[k]
			j2 := FacePointC[k]
			i3 := projected_vertex_x[l]
			l3 := projected_vertex_x[k1]
			k4 := projected_vertex_x[j2]
			if flag && (i3 == -5000 || l3 == -5000 || k4 == -5000) {
				outOfReach[k] = true
				j5 := (projected_vertex_z[l]+projected_vertex_z[k1]+projected_vertex_z[j2])/3 + rcvr.diagonal3DAboveOrigin
				faceLists[j5][++depthListIndices[j5]] = k
			} else {
				if flag1 && rcvr.method486(MouseX, MouseY, projected_vertex_y[l], projected_vertex_y[k1], projected_vertex_y[j2], i3, l3, k4) {
					anIntArray1688[++AnInt1687] = i
					flag1 = false
				}
				if (i3-l3)*(projected_vertex_y[j2]-projected_vertex_y[k1])-(projected_vertex_y[l]-projected_vertex_y[k1])*(k4-l3) > 0 {
					outOfReach[k] = false
					if i3 < 0 || l3 < 0 || k4 < 0 || i3 > Rasterizer2D.lastX || l3 > Rasterizer2D.lastX || k4 > Rasterizer2D.lastX {
						hasAnEdgeToRestrict[k] = true
					} else {
						hasAnEdgeToRestrict[k] = false
					}
					k5 := (projected_vertex_z[l]+projected_vertex_z[k1]+projected_vertex_z[j2])/3 + rcvr.diagonal3DAboveOrigin
					faceLists[k5][++depthListIndices[k5]] = k
				}
			}
		}
	}
	if rcvr.faceRenderPriorities == nil {
		for i1 := rcvr.maxRenderDepth - 1; i1 >= 0; i1-- {
			l1 := depthListIndices[i1]
			if l1 > 0 {
				ai := faceLists[i1]
				for j3 := 0; j3 < l1; j3++ {
					rcvr.method484(ai[j3])
				}
			}
		}
		return
	}
	for j1 := 0; j1 < 12; j1++ {
		anIntArray1673[j1] = 0
		anIntArray1677[j1] = 0
	}
	for i2 := rcvr.maxRenderDepth - 1; i2 >= 0; i2-- {
		k2 := depthListIndices[i2]
		if k2 > 0 {
			ai1 := faceLists[i2]
			for i4 := 0; i4 < k2; i4++ {
				l4 := ai1[i4]
				l5 := faceRenderPriorities[l4]
				j6 := ++anIntArray1673[l5]
				anIntArrayArray1674[l5][j6] = l4
				if l5 < 10 {
					anIntArray1677[l5] += i2
				} else if l5 == 10 {
					anIntArray1675[j6] = i2
				} else {
					anIntArray1676[j6] = i2
				}
			}
		}
	}
	l2 := 0
	if anIntArray1673[1] > 0 || anIntArray1673[2] > 0 {
		l2 = (anIntArray1677[1] + anIntArray1677[2]) / (anIntArray1673[1] + anIntArray1673[2])
	}
	k3 := 0
	if anIntArray1673[3] > 0 || anIntArray1673[4] > 0 {
		k3 = (anIntArray1677[3] + anIntArray1677[4]) / (anIntArray1673[3] + anIntArray1673[4])
	}
	j4 := 0
	if anIntArray1673[6] > 0 || anIntArray1673[8] > 0 {
		j4 = (anIntArray1677[6] + anIntArray1677[8]) / (anIntArray1673[6] + anIntArray1673[8])
	}
	i6 := 0
	k6 := anIntArray1673[10]
	ai2 := anIntArrayArray1674[10]
	ai3 := anIntArray1675
	if i6 == k6 {
		i6 = 0
		k6 = anIntArray1673[11]
		ai2 = anIntArrayArray1674[11]
		ai3 = anIntArray1676
	}
	var i5 int
	if i6 < k6 {
		i5 = ai3[i6]
	} else {
		i5 = -1000
	}
	for l6 := 0; l6 < 10; l6++ {
		for l6 == 0 && i5 > l2 {
			method484(ai2[++i6])
			if i6 == k6 && ai2 != anIntArrayArray1674[11] {
				i6 = 0
				k6 = anIntArray1673[11]
				ai2 = anIntArrayArray1674[11]
				ai3 = anIntArray1676
			}
			if i6 < k6 {
				i5 = ai3[i6]
			} else {
				i5 = -1000
			}
		}
		for l6 == 3 && i5 > k3 {
			method484(ai2[++i6])
			if i6 == k6 && ai2 != anIntArrayArray1674[11] {
				i6 = 0
				k6 = anIntArray1673[11]
				ai2 = anIntArrayArray1674[11]
				ai3 = anIntArray1676
			}
			if i6 < k6 {
				i5 = ai3[i6]
			} else {
				i5 = -1000
			}
		}
		for l6 == 5 && i5 > j4 {
			method484(ai2[++i6])
			if i6 == k6 && ai2 != anIntArrayArray1674[11] {
				i6 = 0
				k6 = anIntArray1673[11]
				ai2 = anIntArrayArray1674[11]
				ai3 = anIntArray1676
			}
			if i6 < k6 {
				i5 = ai3[i6]
			} else {
				i5 = -1000
			}
		}
		i7 := anIntArray1673[l6]
		ai4 := anIntArrayArray1674[l6]
		for j7 := 0; j7 < i7; j7++ {
			method484(ai4[j7])
		}
	}
	for i5 != -1000 {
		method484(ai2[++i6])
		if i6 == k6 && ai2 != anIntArrayArray1674[11] {
			i6 = 0
			ai2 = anIntArrayArray1674[11]
			k6 = anIntArray1673[11]
			ai3 = anIntArray1676
		}
		if i6 < k6 {
			i5 = ai3[i6]
		} else {
			i5 = -1000
		}
	}
}
func (rcvr *Model) method484(i int) {
	if outOfReach[i] {
		rcvr.method485(i)
		return
	}
	j := FacePointA[i]
	k := FacePointB[i]
	l := FacePointC[i]
	Rasterizer3D.textureOutOfDrawingBounds = hasAnEdgeToRestrict[i]
	if rcvr.faceAlpha == nil {
		Rasterizer3D.alpha = 0
	} else {
		Rasterizer3D.alpha = faceAlpha[i]
	}
	var type int
	if rcvr.FaceDrawType == nil {
		type = 0
	} else {
		type = FaceDrawType[i] & 3
	}
	if rcvr.texture != nil && texture[i] != -1 {
		texture_a := j
		texture_b := k
		texture_c := l
		if rcvr.textureCoordinates != nil && textureCoordinates[i] != -1 {
			coordinate := textureCoordinates[i] & 0xff
			texture_a = texturesFaceA[coordinate]
			texture_b = texturesFaceB[coordinate]
			texture_c = texturesFaceC[coordinate]
		}
		if faceHslC[i] == -1 || type == 3 {
			Rasterizer3D.drawTexturedTriangle(projected_vertex_y[j], projected_vertex_y[k], projected_vertex_y[l], projected_vertex_x[j], projected_vertex_x[k], projected_vertex_x[l], faceHslA[i], faceHslA[i], faceHslA[i], anIntArray1668[texture_a], anIntArray1668[texture_b], anIntArray1668[texture_c], camera_vertex_y[texture_a], camera_vertex_y[texture_b], camera_vertex_y[texture_c], camera_vertex_x[texture_a], camera_vertex_x[texture_b], camera_vertex_x[texture_c], texture[i], camera_vertex_z[j], camera_vertex_z[k], camera_vertex_z[l])
		} else {
			Rasterizer3D.drawTexturedTriangle(projected_vertex_y[j], projected_vertex_y[k], projected_vertex_y[l], projected_vertex_x[j], projected_vertex_x[k], projected_vertex_x[l], faceHslA[i], faceHslB[i], faceHslC[i], anIntArray1668[texture_a], anIntArray1668[texture_b], anIntArray1668[texture_c], camera_vertex_y[texture_a], camera_vertex_y[texture_b], camera_vertex_y[texture_c], camera_vertex_x[texture_a], camera_vertex_x[texture_b], camera_vertex_x[texture_c], texture[i], camera_vertex_z[j], camera_vertex_z[k], camera_vertex_z[l])
		}
	} else {
		if type == 0 {
			Rasterizer3D.drawShadedTriangle(projected_vertex_y[j], projected_vertex_y[k], projected_vertex_y[l], projected_vertex_x[j], projected_vertex_x[k], projected_vertex_x[l], faceHslA[i], faceHslB[i], faceHslC[i], camera_vertex_z[j], camera_vertex_z[k], camera_vertex_z[l])
			return
		}
		if type == 1 {
			Rasterizer3D.drawFlatTriangle(projected_vertex_y[j], projected_vertex_y[k], projected_vertex_y[l], projected_vertex_x[j], projected_vertex_x[k], projected_vertex_x[l], modelIntArray3[faceHslA[i]], camera_vertex_z[j], camera_vertex_z[k], camera_vertex_z[l])
			return
		}
	}
}
func (rcvr *Model) method485(i int) {
	j := Rasterizer3D.originViewX
	k := Rasterizer3D.originViewY
	l := 0
	i1 := FacePointA[i]
	j1 := FacePointB[i]
	k1 := FacePointC[i]
	l1 := camera_vertex_x[i1]
	i2 := camera_vertex_x[j1]
	j2 := camera_vertex_x[k1]
	if l1 >= 50 {
		anIntArray1678[l] = projected_vertex_x[i1]
		anIntArray1679[l] = projected_vertex_y[i1]
		anIntArray1680[++l] = faceHslA[i]
	} else {
		k2 := anIntArray1668[i1]
		k3 := camera_vertex_y[i1]
		k4 := faceHslA[i]
		if j2 >= 50 {
			k5 := (50 - l1) * modelIntArray4[j2-l1]
			anIntArray1678[l] = j + (k2+uint32((anIntArray1668[k1]-k2)*k5)>>16)<<SceneGraph.viewDistance/50
			anIntArray1679[l] = k + (k3+uint32((camera_vertex_y[k1]-k3)*k5)>>16)<<SceneGraph.viewDistance/50
			anIntArray1680[++l] = k4 + uint32((faceHslC[i]-k4)*k5)>>16
		}
		if i2 >= 50 {
			l5 := (50 - l1) * modelIntArray4[i2-l1]
			anIntArray1678[l] = j + (k2+uint32((anIntArray1668[j1]-k2)*l5)>>16)<<SceneGraph.viewDistance/50
			anIntArray1679[l] = k + (k3+uint32((camera_vertex_y[j1]-k3)*l5)>>16)<<SceneGraph.viewDistance/50
			anIntArray1680[++l] = k4 + uint32((faceHslB[i]-k4)*l5)>>16
		}
	}
	if i2 >= 50 {
		anIntArray1678[l] = projected_vertex_x[j1]
		anIntArray1679[l] = projected_vertex_y[j1]
		anIntArray1680[++l] = faceHslB[i]
	} else {
		l2 := anIntArray1668[j1]
		l3 := camera_vertex_y[j1]
		l4 := faceHslB[i]
		if l1 >= 50 {
			i6 := (50 - i2) * modelIntArray4[l1-i2]
			anIntArray1678[l] = j + (l2+uint32((anIntArray1668[i1]-l2)*i6)>>16)<<SceneGraph.viewDistance/50
			anIntArray1679[l] = k + (l3+uint32((camera_vertex_y[i1]-l3)*i6)>>16)<<SceneGraph.viewDistance/50
			anIntArray1680[++l] = l4 + uint32((faceHslA[i]-l4)*i6)>>16
		}
		if j2 >= 50 {
			j6 := (50 - i2) * modelIntArray4[j2-i2]
			anIntArray1678[l] = j + (l2+uint32((anIntArray1668[k1]-l2)*j6)>>16)<<SceneGraph.viewDistance/50
			anIntArray1679[l] = k + (l3+uint32((camera_vertex_y[k1]-l3)*j6)>>16)<<SceneGraph.viewDistance/50
			anIntArray1680[++l] = l4 + uint32((faceHslC[i]-l4)*j6)>>16
		}
	}
	if j2 >= 50 {
		anIntArray1678[l] = projected_vertex_x[k1]
		anIntArray1679[l] = projected_vertex_y[k1]
		anIntArray1680[++l] = faceHslC[i]
	} else {
		i3 := anIntArray1668[k1]
		i4 := camera_vertex_y[k1]
		i5 := faceHslC[i]
		if i2 >= 50 {
			k6 := (50 - j2) * modelIntArray4[i2-j2]
			anIntArray1678[l] = j + (i3+uint32((anIntArray1668[j1]-i3)*k6)>>16)<<SceneGraph.viewDistance/50
			anIntArray1679[l] = k + (i4+uint32((camera_vertex_y[j1]-i4)*k6)>>16)<<SceneGraph.viewDistance/50
			anIntArray1680[++l] = i5 + uint32((faceHslB[i]-i5)*k6)>>16
		}
		if l1 >= 50 {
			l6 := (50 - j2) * modelIntArray4[l1-j2]
			anIntArray1678[l] = j + (i3+uint32((anIntArray1668[i1]-i3)*l6)>>16)<<SceneGraph.viewDistance/50
			anIntArray1679[l] = k + (i4+uint32((camera_vertex_y[i1]-i4)*l6)>>16)<<SceneGraph.viewDistance/50
			anIntArray1680[++l] = i5 + uint32((faceHslA[i]-i5)*l6)>>16
		}
	}
	j3 := anIntArray1678[0]
	j4 := anIntArray1678[1]
	j5 := anIntArray1678[2]
	i7 := anIntArray1679[0]
	j7 := anIntArray1679[1]
	k7 := anIntArray1679[2]
	if (j3-j4)*(k7-j7)-(i7-j7)*(j5-j4) > 0 {
		Rasterizer3D.textureOutOfDrawingBounds = false
		texture_a := i1
		texture_b := j1
		texture_c := k1
		if l == 3 {
			if j3 < 0 || j4 < 0 || j5 < 0 || j3 > Rasterizer2D.lastX || j4 > Rasterizer2D.lastX || j5 > Rasterizer2D.lastX {
				Rasterizer3D.textureOutOfDrawingBounds = true
			}
			var l7 int
			if rcvr.FaceDrawType == nil {
				l7 = 0
			} else {
				l7 = FaceDrawType[i] & 3
			}
			if rcvr.texture != nil && texture[i] != -1 {
				if rcvr.textureCoordinates != nil && textureCoordinates[i] != -1 {
					coordinate := textureCoordinates[i] & 0xff
					texture_a = texturesFaceA[coordinate]
					texture_b = texturesFaceB[coordinate]
					texture_c = texturesFaceC[coordinate]
				}
				if faceHslC[i] == -1 {
					Rasterizer3D.drawTexturedTriangle(i7, j7, k7, j3, j4, j5, faceHslA[i], faceHslA[i], faceHslA[i], anIntArray1668[texture_a], anIntArray1668[texture_b], anIntArray1668[texture_c], camera_vertex_y[texture_a], camera_vertex_y[texture_b], camera_vertex_y[texture_c], camera_vertex_x[texture_a], camera_vertex_x[texture_b], camera_vertex_x[texture_c], texture[i], camera_vertex_z[i1], camera_vertex_z[j1], camera_vertex_z[k1])
				} else {
					Rasterizer3D.drawTexturedTriangle(i7, j7, k7, j3, j4, j5, anIntArray1680[0], anIntArray1680[1], anIntArray1680[2], anIntArray1668[texture_a], anIntArray1668[texture_b], anIntArray1668[texture_c], camera_vertex_y[texture_a], camera_vertex_y[texture_b], camera_vertex_y[texture_c], camera_vertex_x[texture_a], camera_vertex_x[texture_b], camera_vertex_x[texture_c], texture[i], camera_vertex_z[i1], camera_vertex_z[j1], camera_vertex_z[k1])
				}
			} else {
				if l7 == 0 {
					Rasterizer3D.drawShadedTriangle(i7, j7, k7, j3, j4, j5, anIntArray1680[0], anIntArray1680[1], anIntArray1680[2], -1f, -1f, -1f)
				} else if l7 == 1 {
					Rasterizer3D.drawFlatTriangle(i7, j7, k7, j3, j4, j5, modelIntArray3[faceHslA[i]], -1f, -1f, -1f)
				}
			}
		}
		if l == 4 {
			if j3 < 0 || j4 < 0 || j5 < 0 || j3 > Rasterizer2D.lastX || j4 > Rasterizer2D.lastX || j5 > Rasterizer2D.lastX || anIntArray1678[3] < 0 || anIntArray1678[3] > Rasterizer2D.lastX {
				Rasterizer3D.textureOutOfDrawingBounds = true
			}
			var type int
			if rcvr.FaceDrawType == nil {
				type = 0
			} else {
				type = FaceDrawType[i] & 3
			}
			if rcvr.texture != nil && texture[i] != -1 {
				if rcvr.textureCoordinates != nil && textureCoordinates[i] != -1 {
					coordinate := textureCoordinates[i] & 0xff
					texture_a = texturesFaceA[coordinate]
					texture_b = texturesFaceB[coordinate]
					texture_c = texturesFaceC[coordinate]
				}
				if faceHslC[i] == -1 {
					Rasterizer3D.drawTexturedTriangle(i7, j7, k7, j3, j4, j5, faceHslA[i], faceHslA[i], faceHslA[i], anIntArray1668[texture_a], anIntArray1668[texture_b], anIntArray1668[texture_c], camera_vertex_y[texture_a], camera_vertex_y[texture_b], camera_vertex_y[texture_c], camera_vertex_x[texture_a], camera_vertex_x[texture_b], camera_vertex_x[texture_c], texture[i], camera_vertex_z[i1], camera_vertex_z[j1], camera_vertex_z[k1])
					Rasterizer3D.drawTexturedTriangle(i7, k7, anIntArray1679[3], j3, j5, anIntArray1678[3], faceHslA[i], faceHslA[i], faceHslA[i], anIntArray1668[texture_a], anIntArray1668[texture_b], anIntArray1668[texture_c], camera_vertex_y[texture_a], camera_vertex_y[texture_b], camera_vertex_y[texture_c], camera_vertex_x[texture_a], camera_vertex_x[texture_b], camera_vertex_x[texture_c], texture[i], camera_vertex_z[i1], camera_vertex_z[j1], camera_vertex_z[k1])
				} else {
					Rasterizer3D.drawTexturedTriangle(i7, j7, k7, j3, j4, j5, anIntArray1680[0], anIntArray1680[1], anIntArray1680[2], anIntArray1668[texture_a], anIntArray1668[texture_b], anIntArray1668[texture_c], camera_vertex_y[texture_a], camera_vertex_y[texture_b], camera_vertex_y[texture_c], camera_vertex_x[texture_a], camera_vertex_x[texture_b], camera_vertex_x[texture_c], texture[i], camera_vertex_z[i1], camera_vertex_z[j1], camera_vertex_z[k1])
					Rasterizer3D.drawTexturedTriangle(i7, k7, anIntArray1679[3], j3, j5, anIntArray1678[3], anIntArray1680[0], anIntArray1680[2], anIntArray1680[3], anIntArray1668[texture_a], anIntArray1668[texture_b], anIntArray1668[texture_c], camera_vertex_y[texture_a], camera_vertex_y[texture_b], camera_vertex_y[texture_c], camera_vertex_x[texture_a], camera_vertex_x[texture_b], camera_vertex_x[texture_c], texture[i], camera_vertex_z[i1], camera_vertex_z[j1], camera_vertex_z[k1])
					return
				}
			} else {
				if type == 0 {
					Rasterizer3D.drawShadedTriangle(i7, j7, k7, j3, j4, j5, anIntArray1680[0], anIntArray1680[1], anIntArray1680[2], -1f, -1f, -1f)
					Rasterizer3D.drawShadedTriangle(i7, k7, anIntArray1679[3], j3, j5, anIntArray1678[3], anIntArray1680[0], anIntArray1680[2], anIntArray1680[3], camera_vertex_z[i1], camera_vertex_z[j1], camera_vertex_z[k1])
					return
				}
				if type == 1 {
					l8 := modelIntArray3[faceHslA[i]]
					Rasterizer3D.drawFlatTriangle(i7, j7, k7, j3, j4, j5, l8, -1f, -1f, -1f)
					Rasterizer3D.drawFlatTriangle(i7, k7, anIntArray1679[3], j3, j5, anIntArray1678[3], l8, camera_vertex_z[i1], camera_vertex_z[j1], camera_vertex_z[k1])
					return
				}
			}
		}
	}
}
func (rcvr *Model) method486(i int, j int, k int, l int, i1 int, j1 int, k1 int, l1 int) (bool) {
	if j < k && j < l && j < i1 {
		return false
	}
	if j > k && j > l && j > i1 {
		return false
	}
	if i < j1 && i < k1 && i < l1 {
		return false
	}
	return i <= j1 || i <= k1 || i <= l1
}
func (rcvr *Model) Recolor(found int, replace int) {
	if rcvr.triangleColours != nil {
		for face := 0; face < rcvr.TriangleCount; face++ {
			if triangleColours[face] == found.(int16) {
				triangleColours[face] = replace.(int16)
			}
		}
	}
}
func (rcvr *Model) RenderAtPoint(i int, j int, k int, l int, i1 int, j1 int, k1 int, l1 int, i2 int) {
	j2 := uint32(l1*i1-j1*l) >> 16
	k2 := uint32(k1*j+j2*k) >> 16
	l2 := uint32(rcvr.MaxVertexDistanceXZPlane*k) >> 16
	i3 := k2 + l2
	if i3 <= 50 || k2 >= 3500 {
		return
	}
	j3 := uint32(l1*l+j1*i1) >> 16
	k3 := (j3 - rcvr.MaxVertexDistanceXZPlane) << SceneGraph.viewDistance
	if k3/i3 >= Rasterizer2D.viewportCenterX {
		return
	}
	l3 := (j3 + rcvr.MaxVertexDistanceXZPlane) << SceneGraph.viewDistance
	if l3/i3 <= -Rasterizer2D.viewportCenterX {
		return
	}
	i4 := uint32(k1*k-j2*j) >> 16
	j4 := uint32(rcvr.MaxVertexDistanceXZPlane*j) >> 16
	k4 := (i4 + j4) << SceneGraph.viewDistance
	if k4/i3 <= -Rasterizer2D.viewportCenterY {
		return
	}
	l4 := j4 + uint32(<<super>>*k)>>16
	i5 := (i4 - l4) << SceneGraph.viewDistance
	if i5/i3 >= Rasterizer2D.viewportCenterY {
		return
	}
	j5 := l2 + uint32(<<super>>*j)>>16
	flag := false
	if k2-j5 <= 50 {
		flag = true
	}
	flag1 := false
	if i2 > 0 && ABoolean1684 {
		k5 := k2 - l2
		if k5 <= 50 {
			k5 = 50
		}
		if j3 > 0 {
			k3 /= i3
			l3 /= k5
		} else {
			l3 /= i3
			k3 /= k5
		}
		if i4 > 0 {
			i5 /= i3
			k4 /= k5
		} else {
			k4 /= i3
			i5 /= k5
		}
		i6 := MouseX - Rasterizer3D.originViewX
		k6 := MouseY - Rasterizer3D.originViewY
		if i6 > k3 && i6 < l3 && k6 > i5 && k6 < k4 {
			if rcvr.fitsOnSingleTile {
				anIntArray1688[++AnInt1687] = i2
			} else {
				flag1 = true
			}
		}
	}
	l5 := Rasterizer3D.originViewX
	j6 := Rasterizer3D.originViewY
	l6 := 0
	i7 := 0
	if i != 0 {
		l6 = Sine[i]
		i7 = Cosine[i]
	}
	for j7 := 0; j7 < rcvr.VerticeCount; j7++ {
		k7 := VertexX[j7]
		l7 := VertexY[j7]
		i8 := VertexZ[j7]
		if i != 0 {
			j8 := uint32(i8*l6+k7*i7) >> 16
			i8 = uint32(i8*i7-k7*l6) >> 16
			k7 = j8
		}
		k7 += j1
		l7 += k1
		i8 += l1
		k8 := uint32(i8*l+k7*i1) >> 16
		i8 = uint32(i8*i1-k7*l) >> 16
		k7 = k8
		k8 = uint32(l7*k-i8*j) >> 16
		i8 = uint32(l7*j+i8*k) >> 16
		l7 = k8
		projected_vertex_z[j7] = i8 - k2
		camera_vertex_z[j7] = i8
		if i8 >= 50 {
			projected_vertex_x[j7] = l5 + k7<<SceneGraph.viewDistance/i8
			projected_vertex_y[j7] = j6 + l7<<SceneGraph.viewDistance/i8
		} else {
			projected_vertex_x[j7] = -5000
			flag = true
		}
		if flag || rcvr.textureTriangleCount > 0 {
			anIntArray1668[j7] = k7
			camera_vertex_y[j7] = l7
			camera_vertex_x[j7] = i8
		}
	}
	if try() {
		rcvr.method483(flag, flag1, i2)
		return
	} else if catch_Exception(_ex) {
		return
	}
}
func (rcvr *Model) Retexture(found int16, replace int16) {
	if rcvr.texture != nil {
		for face := 0; face < rcvr.TriangleCount; face++ {
			if texture[face] == found {
				texture[face] = replace
			}
		}
	}
}
func (rcvr *Model) Rotate90Degrees() {
	for index := 0; index < rcvr.VerticeCount; index++ {
		x := VertexX[index]
		VertexX[index] = VertexZ[index]
		VertexZ[index] = -x
	}
}
func (rcvr *Model) Scale(x int, z int, y int) {
	for index := 0; index < rcvr.VerticeCount; index++ {
		VertexX[index] = VertexX[index] * x / 128
		VertexY[index] = VertexY[index] * y / 128
		VertexZ[index] = VertexZ[index] * z / 128
	}
}
func (rcvr *Model) Translate(x int, y int, z int) {
	for index := 0; index < rcvr.VerticeCount; index++ {
		VertexX[index] += x
		VertexY[index] += y
		VertexZ[index] += z
	}
}

var Models = NewReferenceCache(30)
var BaseModels = NewReferenceCache(500)
var modelSegments = make([]*Model, 4)
var cache []*ObjectDefinition
var stream *Buffer
var streamIndices []int
var cacheIndex int
var Length int

type ObjectDefinition struct {
	id                   int
	modelIds             []int
	modelTypes           []int
	ChildrenIds          []int
	name                 string
	originalModelTexture []int16
	modifiedModelTexture []int16
	originalModelColors  []int
	modifiedModelColors  []int
	ObjectSizeX          int
	ObjectSizeY          int
	scaleX               int
	scaleY               int
	scaleZ               int
	translateX           int
	translateY           int
	translateZ           int
	Solid                bool
	Walkable             bool
	IsInteractive        bool
	ContouredGround      bool
	delayShading         bool
	Occludes             bool
	inverted             bool
	CastsShadow          bool
	ObstructsGround      bool
	removeClipping       bool
	interactions         []string
	ambientLighting      byte
	lightDiffusion       int
	DecorDisplacement    int
}

func NewObjectDefinition() (rcvr *ObjectDefinition) {
	rcvr = &ObjectDefinition{}
	rcvr.id = -1
	return
}
func (rcvr *ObjectDefinition) Decode(buffer *Buffer) {
	for true {
		opcode := buffer.readUnsignedByte()
		if opcode == 0 {
			break
		} else if opcode == 1 {
			length := buffer.readUnsignedByte()
			if length > 0 {
				objectTypes := make([]int, length)
				objectModels := make([]int, length)
				for index := 0; index < length; index++ {
					objectModels[index] = buffer.readUShort()
					objectTypes[index] = buffer.readUnsignedByte()
				}
				rcvr.modelTypes = objectTypes
				rcvr.modelIds = objectModels
			}
		} else if opcode == 2 {
			rcvr.name = buffer.readString()
		} else if opcode == 5 {
			length := buffer.readUnsignedByte()
			if length > 0 {
				rcvr.modelTypes = nil
				rcvr.modelIds = make([]int, length)
				for index := 0; index < length; index++ {
					modelIds[index] = buffer.readUShort()
				}
			}
		} else if opcode == 14 {
			rcvr.ObjectSizeX = buffer.readUnsignedByte()
		} else if opcode == 15 {
			rcvr.ObjectSizeY = buffer.readUnsignedByte()
		} else if opcode == 17 {
			rcvr.Solid = false
			rcvr.Walkable = false
		} else if opcode == 18 {
			rcvr.Walkable = false
		} else if opcode == 19 {
			rcvr.IsInteractive = buffer.readUnsignedByte() == 1
		} else if opcode == 21 {
			rcvr.ContouredGround = true
		} else if opcode == 22 {
			rcvr.delayShading = true
		} else if opcode == 23 {
			rcvr.Occludes = true
		} else if opcode == 24 {
			animation := buffer.readUShort()
		} else if opcode == 27 {
		} else if opcode == 28 {
			rcvr.DecorDisplacement = buffer.readUnsignedByte()
		} else if opcode == 29 {
			rcvr.ambientLighting = buffer.readSignedByte()
		} else if opcode == 39 {
			rcvr.lightDiffusion = buffer.readSignedByte() * 25
		} else if opcode >= 30 && opcode < 35 {
			if rcvr.interactions == nil {
				rcvr.interactions = make([]string, 5)
			}
			interactions[opcode-30] = buffer.readString()
			if interactions[opcode-30].equalsIgnoreCase("Hidden") {
				interactions[opcode-30] = nil
			}
		} else if opcode == 40 {
			length := buffer.readUnsignedByte()
			rcvr.modifiedModelColors = make([]int, length)
			rcvr.originalModelColors = make([]int, length)
			for index := 0; index < length; index++ {
				modifiedModelColors[index] = buffer.readUShort()
				originalModelColors[index] = buffer.readUShort()
			}
		} else if opcode == 41 {
			length := buffer.readUnsignedByte()
			rcvr.modifiedModelTexture = make([]int16, length)
			rcvr.originalModelTexture = make([]int16, length)
			for index := 0; index < length; index++ {
				modifiedModelTexture[index] = buffer.readUShort().(int16)
				originalModelTexture[index] = buffer.readUShort().(int16)
			}
		} else if opcode == 62 {
			rcvr.inverted = true
		} else if opcode == 64 {
			rcvr.CastsShadow = false
		} else if opcode == 65 {
			rcvr.scaleX = buffer.readUShort()
		} else if opcode == 66 {
			rcvr.scaleY = buffer.readUShort()
		} else if opcode == 67 {
			rcvr.scaleZ = buffer.readUShort()
		} else if opcode == 68 {
			mapscene := buffer.readUShort()
		} else if opcode == 69 {
			surroundings := buffer.readUnsignedByte()
		} else if opcode == 70 {
			rcvr.translateX = buffer.readUShort()
		} else if opcode == 71 {
			rcvr.translateY = buffer.readUShort()
		} else if opcode == 72 {
			rcvr.translateZ = buffer.readUShort()
		} else if opcode == 73 {
			rcvr.ObstructsGround = true
		} else if opcode == 74 {
			rcvr.removeClipping = true
		} else if opcode == 75 {
			supportItems := buffer.readUnsignedByte()
		} else if opcode == 77 {
			varpId := buffer.readUShort()
			configId := buffer.readUShort()
			length := buffer.readUnsignedByte()
			configChangeDest := make([]int, length+2)
			for index := 0; index <= length; index++ {
				configChangeDest[index] = buffer.readUShort()
				if 0xFFFF == configChangeDest[index] {
					configChangeDest[index] = -1
				}
			}
		} else if opcode == 78 {
			buffer.readUShort()
			buffer.readUnsignedByte()
		} else if opcode == 79 {
			buffer.readUShort()
			buffer.readUShort()
			buffer.readUnsignedByte()
			length := buffer.readUnsignedByte()
			anIntArray2084 := make([]int, length)
			for index := 0; index < length; index++ {
				anIntArray2084[index] = buffer.readUShort()
			}
		} else if opcode == 81 {
			buffer.readUnsignedByte()
		} else if opcode == 82 {
			minimapFunction := buffer.readUShort()
		} else if opcode == 92 {
			varpId := buffer.readUShort()
			configId := buffer.readUShort()
			var := buffer.readUShort()
			length := buffer.readUnsignedByte()
			configChangeDest := make([]int, length+2)
			for index := 0; index <= length; index++ {
				configChangeDest[index] = buffer.readUShort()
				if 0xFFFF == configChangeDest[index] {
					configChangeDest[index] = -1
				}
			}
		} else if opcode == 249 {
			length := buffer.readUnsignedByte()
			params := NewHashMap(length)
			for i := 0; i < length; i++ {
				isString := buffer.readUnsignedByte() == 1
				key := buffer.read24Int()
				var value *Object
				if isString {
					value = buffer.readString()
				} else {
					value = buffer.readInt()
				}
				params.put(key, value)
			}
		} else {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("%v%v", "invalid opcode: ", opcode))
		}
	}
	rcvr.postDecode()
}
func (rcvr *ObjectDefinition) Deconstruct() {
	rcvr.modelIds = nil
	rcvr.modelTypes = nil
	rcvr.name = nil
	rcvr.modifiedModelColors = nil
	rcvr.originalModelColors = nil
	rcvr.modifiedModelTexture = nil
	rcvr.originalModelTexture = nil
	rcvr.ObjectSizeX = 1
	rcvr.ObjectSizeY = 1
	rcvr.Solid = true
	rcvr.Walkable = true
	rcvr.IsInteractive = false
	rcvr.ContouredGround = false
	rcvr.delayShading = false
	rcvr.Occludes = false
	rcvr.DecorDisplacement = 16
	rcvr.ambientLighting = 0
	rcvr.lightDiffusion = 0
	rcvr.interactions = nil
	rcvr.inverted = false
	rcvr.CastsShadow = true
	rcvr.scaleX = 128
	rcvr.scaleY = 128
	rcvr.scaleZ = 128
	rcvr.translateX = 0
	rcvr.translateY = 0
	rcvr.translateZ = 0
	rcvr.ObstructsGround = false
	rcvr.removeClipping = false
	rcvr.ChildrenIds = nil
}
func Get(id int) (*ObjectDefinition) {
	if id > len(streamIndices) {
		id = len(streamIndices) - 1
	}
	for index := 0; index < 20; index++ {
		if <<unimp_obj.nm_*parser.GoArrayReference>> == id {
			return cache[index]
		}
	}
	cacheIndex = (cacheIndex + 1) % 20
	objectDef := cache[cacheIndex]
	stream.currentPosition = streamIndices[id]
	objectDef.id = id
	objectDef.deconstruct()
	objectDef.decode(stream)
	return objectDef
}
func Initialize(archive *FileArchive) {
	stream = NewBuffer(archive.readFile("loc.dat"))
	stream := NewBuffer(archive.readFile("loc.idx"))
	Length = stream.readUShort()
	streamIndices = make([]int, Length)
	offset := 2
	for index := 0; index < Length; index++ {
		streamIndices[index] = offset
		offset += stream.readUShort()
	}
	cache = make([]*ObjectDefinition, 20)
	for index := 0; index < 20; index++ {
		cache[index] = NewObjectDefinition()
	}
}
func (rcvr *ObjectDefinition) Model(type int, orientation int) (*Model) {
	model := nil
	var key int64
	if rcvr.modelTypes == nil {
		if type != 10 {
			return nil
		}
		key = (rcvr.id<<6 + orientation).(int64)
		cached, ok := Models.get(key).(*Model)
		if !ok {
			panic("XXX Cast fail for *parser.GoCastType")
		}
		if cached != nil {
			return cached
		}
		if rcvr.modelIds == nil {
			return nil
		}
		invert := rcvr.inverted ^ (orientation > 3)
		length := len(rcvr.modelIds)
		for index := 0; index < length; index++ {
			invertId := modelIds[index]
			if invert {
				invertId += 0x10000
			}
			model = BaseModels.get(invertId).(*Model)
			if model == nil {
				model = Model.Get(invertId & 0xffff)
				if model == nil {
					return nil
				}
				if invert {
					model.invert()
				}
				BaseModels.put(model, invertId)
			}
			if length > 1 {
				modelSegments[index] = model
			}
		}
		if length > 1 {
			model = NewModel2(length, modelSegments)
		}
	} else {
		modelId := -1
		for index := 0; index < len(rcvr.modelTypes); index++ {
			if modelTypes[index] != type {
				continue
			}
			modelId = index
			break
		}
		if modelId == -1 {
			return nil
		}
		key = (rcvr.id<<8 + modelId<<3 + orientation).(int64)
		cached, ok := Models.get(key).(*Model)
		if !ok {
			panic("XXX Cast fail for *parser.GoCastType")
		}
		if cached != nil {
			return cached
		}
		if rcvr.modelIds == nil {
			return nil
		}
		modelId = modelIds[modelId]
		invert := rcvr.inverted ^ (orientation > 3)
		if invert {
			modelId += 0x10000
		}
		model = BaseModels.get(modelId).(*Model)
		if model == nil {
			model = Model.Get(modelId & 0xffff)
			if model == nil {
				return nil
			}
			if invert {
				model.invert()
			}
			BaseModels.put(model, modelId)
		}
	}
	scale := rcvr.scaleX != 128 || rcvr.scaleY != 128 || rcvr.scaleZ != 128
	translate := rcvr.translateX != 0 || rcvr.translateY != 0 || rcvr.translateZ != 0
	cached := NewModel(rcvr.modifiedModelColors == nil, true, orientation == 0 && !scale && !translate, rcvr.modifiedModelTexture == nil, model)
	for --orientation > 0 {
		cached.rotate90Degrees()
	}
	if rcvr.modifiedModelColors != nil {
		for k2 := 0; k2 < len(rcvr.modifiedModelColors); k2++ {
			cached.recolor(modifiedModelColors[k2], originalModelColors[k2])
		}
	}
	if rcvr.modifiedModelTexture != nil {
		for k2 := 0; k2 < len(rcvr.modifiedModelTexture); k2++ {
			cached.retexture(modifiedModelTexture[k2], originalModelTexture[k2])
		}
	}
	if scale {
		cached.scale(rcvr.scaleX, rcvr.scaleZ, rcvr.scaleY)
	}
	if translate {
		cached.translate(rcvr.translateX, rcvr.translateY, rcvr.translateZ)
	}
	cached.light(85+rcvr.ambientLighting, 768+rcvr.lightDiffusion, -50, -10, -50, !rcvr.delayShading)
	Models.put(cached, key)
	return cached
}
func (rcvr *ObjectDefinition) ModelAt(type int, orientation int, cosineY int, sineY int, cosineX int, sineX int) (*Model) {
	model := rcvr.Model(type, orientation)
	if model == nil {
		return nil
	}
	if rcvr.ContouredGround || rcvr.delayShading {
		model = NewModel(rcvr.ContouredGround, rcvr.delayShading, model)
	}
	if rcvr.ContouredGround {
		y := (cosineY + sineY + cosineX + sineX) / 4
		for vertex := 0; vertex < model.verticeCount; vertex++ {
			startX := model[vertex]
			startZ := model[vertex]
			z := cosineY + (sineY-cosineY)*(startX+64)/128
			x := sineX + (cosineX-sineX)*(startX+64)/128
			undulationOffset := z + (x-z)*(startZ+64)/128
			model[vertex] += undulationOffset - y
		}
		model.computeSphericalBounds()
	}
	return model
}
func (rcvr *ObjectDefinition) postDecode() {
	if rcvr.name != nil && !rcvr.name.equals("null") {
		rcvr.IsInteractive = rcvr.modelIds != nil && (rcvr.modelTypes == nil || modelTypes[0] == 10) || rcvr.interactions != nil
	}
	if rcvr.removeClipping {
		rcvr.Solid = false
		rcvr.Walkable = false
	}
}

var pixels []int
var width int
var height int
var bottomY int
var bottomX int
var LastX int
var ViewportCenterX int
var ViewportCenterY int
var depthBuffer []float32

type Rasterizer2D struct {
	*Cacheable
}

func NewRasterizer2D() (rcvr *Rasterizer2D) {
	rcvr = &Rasterizer2D{}
	return
}
func Clear() {
	i := width * height
	for j := 0; j < i; j++ {
		pixels[j] = 0
		depthBuffer[j] = Float.MAX_VALUE
	}
}
func InitializeDrawingArea(height int, width int, pixels []int, depth []float32) {
	depthBuffer = depth
	Rasterizer2D.pixels = pixels
	Rasterizer2D.width = width
	Rasterizer2D.height = height
	Rasterizer2D.SetDrawingArea(height, width)
}
func SetDrawingArea(bottomY int, rightX int) {
	if rightX > width {
		rightX = width
	}
	if bottomY > height {
		bottomY = height
	}
	bottomX = rightX
	Rasterizer2D.bottomY = bottomY
	LastX = bottomX
	ViewportCenterX = bottomX / 2
	ViewportCenterY = Rasterizer2D.bottomY / 2
}

const tEXTURE_LENGTH = 61

var DEPTH []int
var HslToRgb = make([]int, 0x10000)
var textureRequestBufferPointer int
var textureRequestPixelBuffer []int
var TextureOutOfDrawingBounds bool
var lastTextureRetrievalCount int
var textureIsNotTransparant bool
var Sine []int
var Cosine []int
var OriginViewX int
var OriginViewY int
var textureCount int
var scanOffsets []int
var shadowDecay []int
var Alpha int
var textures = make([]*IndexedImage, tEXTURE_LENGTH)
var texturesPixelBuffer = make([][]int, tEXTURE_LENGTH)
var currentPalette = make([][]int, tEXTURE_LENGTH)
var averageTextureColours = make([]int, tEXTURE_LENGTH)
var textureIsTransparant = make([]bool, tEXTURE_LENGTH)
var textureLastUsed = make([]int, tEXTURE_LENGTH)

type Rasterizer3D struct {
	*Rasterizer2D
}

func NewRasterizer3D() (rcvr *Rasterizer3D) {
	rcvr = &Rasterizer3D{}
	return
}
func adjustBrightness(rgb int, intensity float64) (int) {
	r := uint32(rgb) >> 16 / 256D
	g := uint32(rgb) >> 8 & 0xff / 256D
	b := rgb & 0xff / 256D
	r = Math.pow(r, intensity)
	g = Math.pow(g, intensity)
	b = Math.pow(b, intensity)
	r_byte, ok := (r * 256D).(int)
	if !ok {
		panic("XXX Cast fail for *parser.GoCastType")
	}
	g_byte, ok := (g * 256D).(int)
	if !ok {
		panic("XXX Cast fail for *parser.GoCastType")
	}
	b_byte, ok := (b * 256D).(int)
	if !ok {
		panic("XXX Cast fail for *parser.GoCastType")
	}
	return r_byte<<16 + g_byte<<8 + b_byte
}
func ClearTextureCache() {
	textureRequestPixelBuffer = nil
	for i := 0; i < tEXTURE_LENGTH; i++ {
		texturesPixelBuffer[i] = nil
	}
}
func drawFlatTexturedScanline(dest []int, dest_off int, loops int, start_x int, end_x int, depth float32, depth_slope float32) {
	var rgb int
	if TextureOutOfDrawingBounds {
		if end_x > Rasterizer2D.lastX {
			end_x = Rasterizer2D.lastX
		}
		if start_x < 0 {
			start_x = 0
		}
	}
	if start_x >= end_x {
		return
	}
	dest_off += start_x
	rgb = uint32(end_x-start_x) >> 2
	depth += depth_slope * start_x
	if Alpha == 0 {
		for --rgb >= 0 {
			for i := 0; i < 4; i++ {
				if true {
					dest[dest_off] = loops
					<<unimp_arrayref_Rasterizer2D.depthBuffer>>[dest_off] = depth
				}
				dest_off++
				depth += depth_slope
			}
		}
		for rgb = (end_x-start_x)&3; --rgb >= 0; {
			if true {
				dest[dest_off] = loops
				<<unimp_arrayref_Rasterizer2D.depthBuffer>>[dest_off] = depth
			}
			dest_off++
			depth += depth_slope
		}
		return
	}
	dest_alpha := Alpha
	src_alpha := 256 - Alpha
	loops = uint32(loops&0xff00ff*src_alpha)>>8&0xff00ff + uint32(loops&0xff00*src_alpha)>>8&0xff00
	for --rgb >= 0 {
		for i := 0; i < 4; i++ {
			if true {
				dest[dest_off] = loops + uint32(dest[dest_off]&0xff00ff*dest_alpha)>>8&0xff00ff + uint32(dest[dest_off]&0xff00*dest_alpha)>>8&0xff00
				<<unimp_arrayref_Rasterizer2D.depthBuffer>>[dest_off] = depth
			}
			dest_off++
			depth += depth_slope
		}
	}
	for rgb = (end_x-start_x)&3; --rgb >= 0; {
		if true {
			dest[dest_off] = loops + uint32(dest[dest_off]&0xff00ff*dest_alpha)>>8&0xff00ff + uint32(dest[dest_off]&0xff00*dest_alpha)>>8&0xff00
			<<unimp_arrayref_Rasterizer2D.depthBuffer>>[dest_off] = depth
		}
		dest_off++
		depth += depth_slope
	}
}
func DrawFlatTriangle(y_a int, y_b int, y_c int, x_a int, x_b int, x_c int, k1 int, z_a float32, z_b float32, z_c float32) {
	if z_a < 0 || z_b < 0 || z_c < 0 {
		return
	}
	a_to_b := 0
	if y_b != y_a {
		a_to_b = (x_b - x_a) << 16 / (y_b - y_a)
	}
	b_to_c := 0
	if y_c != y_b {
		b_to_c = (x_c - x_b) << 16 / (y_c - y_b)
	}
	c_to_a := 0
	if y_c != y_a {
		c_to_a = (x_a - x_c) << 16 / (y_a - y_c)
	}
	b_aX := x_b - x_a
	b_aY := y_b - y_a
	c_aX := x_c - x_a
	c_aY := y_c - y_a
	b_aZ := z_b - z_a
	c_aZ := z_c - z_a
	div := b_aX*c_aY - c_aX*b_aY
	depth_slope := (b_aZ*c_aY - c_aZ*b_aY) / div
	depth_increment := (c_aZ*b_aX - b_aZ*c_aX) / div
	if y_a <= y_b && y_a <= y_c {
		if y_a >= Rasterizer2D.bottomY {
			return
		}
		if y_b > Rasterizer2D.bottomY {
			y_b = Rasterizer2D.bottomY
		}
		if y_c > Rasterizer2D.bottomY {
			y_c = Rasterizer2D.bottomY
		}
		z_a = z_a - depth_slope*x_a + depth_slope
		if y_b < y_c {
			x_a = x_a << 16
			x_c = x_a
			if y_a < 0 {
				x_c -= c_to_a * y_a
				x_a -= a_to_b * y_a
				z_a -= depth_increment * y_a
				y_a = 0
			}
			x_b = x_b << 16
			if y_b < 0 {
				x_b -= b_to_c * y_b
				y_b = 0
			}
			if y_a != y_b && c_to_a < a_to_b || y_a == y_b && c_to_a > b_to_c {
				y_c -= y_b
				y_b -= y_a
				for y_a = scanOffsets[y_a]; --y_b >= 0; y_a = Rasterizer2D.width {
					Rasterizer3D.drawFlatTexturedScanline(Rasterizer2D.pixels, y_a, k1, uint32(x_c)>>16, uint32(x_a)>>16, z_a, depth_slope)
					x_c += c_to_a
					x_a += a_to_b
					z_a += depth_increment
				}
				for --y_c >= 0 {
					drawFlatTexturedScanline(Rasterizer2D.pixels, y_a, k1, uint32(x_c)>>16, uint32(x_b)>>16, z_a, depth_slope)
					x_c += c_to_a
					x_b += b_to_c
					y_a += Rasterizer2D.width
					z_a += depth_increment
				}
				return
			}
			y_c -= y_b
			y_b -= y_a
			for y_a = scanOffsets[y_a]; --y_b >= 0; y_a = Rasterizer2D.width {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_a, k1, uint32(x_a)>>16, uint32(x_c)>>16, z_a, depth_slope)
				x_c += c_to_a
				x_a += a_to_b
				z_a += depth_increment
			}
			for --y_c >= 0 {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_a, k1, uint32(x_b)>>16, uint32(x_c)>>16, z_a, depth_slope)
				x_c += c_to_a
				x_b += b_to_c
				y_a += Rasterizer2D.width
				z_a += depth_increment
			}
			return
		}
		x_a = x_a << 16
		x_b = x_a
		if y_a < 0 {
			x_b -= c_to_a * y_a
			x_a -= a_to_b * y_a
			z_a -= depth_increment * y_a
			y_a = 0
		}
		x_c = x_c << 16
		if y_c < 0 {
			x_c -= b_to_c * y_c
			y_c = 0
		}
		if y_a != y_c && c_to_a < a_to_b || y_a == y_c && b_to_c > a_to_b {
			y_b -= y_c
			y_c -= y_a
			for y_a = scanOffsets[y_a]; --y_c >= 0; y_a = Rasterizer2D.width {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_a, k1, uint32(x_b)>>16, uint32(x_a)>>16, z_a, depth_slope)
				z_a += depth_increment
				x_b += c_to_a
				x_a += a_to_b
			}
			for --y_b >= 0 {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_a, k1, uint32(x_c)>>16, uint32(x_a)>>16, z_a, depth_slope)
				z_a += depth_increment
				x_c += b_to_c
				x_a += a_to_b
				y_a += Rasterizer2D.width
			}
			return
		}
		y_b -= y_c
		y_c -= y_a
		for y_a = scanOffsets[y_a]; --y_c >= 0; y_a = Rasterizer2D.width {
			drawFlatTexturedScanline(Rasterizer2D.pixels, y_a, k1, uint32(x_a)>>16, uint32(x_b)>>16, z_a, depth_slope)
			z_a += depth_increment
			x_b += c_to_a
			x_a += a_to_b
		}
		for --y_b >= 0 {
			drawFlatTexturedScanline(Rasterizer2D.pixels, y_a, k1, uint32(x_a)>>16, uint32(x_c)>>16, z_a, depth_slope)
			z_a += depth_increment
			x_c += b_to_c
			x_a += a_to_b
			y_a += Rasterizer2D.width
		}
		return
	}
	if y_b <= y_c {
		if y_b >= Rasterizer2D.bottomY {
			return
		}
		if y_c > Rasterizer2D.bottomY {
			y_c = Rasterizer2D.bottomY
		}
		if y_a > Rasterizer2D.bottomY {
			y_a = Rasterizer2D.bottomY
		}
		z_b = z_b - depth_slope*x_b + depth_slope
		if y_c < y_a {
			x_b = x_b << 16
			x_a = x_b
			if y_b < 0 {
				x_a -= a_to_b * y_b
				x_b -= b_to_c * y_b
				z_b -= depth_increment * y_b
				y_b = 0
			}
			x_c = x_c << 16
			if y_c < 0 {
				x_c -= c_to_a * y_c
				y_c = 0
			}
			if y_b != y_c && a_to_b < b_to_c || y_b == y_c && a_to_b > c_to_a {
				y_a -= y_c
				y_c -= y_b
				for y_b = scanOffsets[y_b]; --y_c >= 0; y_b = Rasterizer2D.width {
					drawFlatTexturedScanline(Rasterizer2D.pixels, y_b, k1, uint32(x_a)>>16, uint32(x_b)>>16, z_b, depth_slope)
					z_b += depth_increment
					x_a += a_to_b
					x_b += b_to_c
				}
				for --y_a >= 0 {
					drawFlatTexturedScanline(Rasterizer2D.pixels, y_b, k1, uint32(x_a)>>16, uint32(x_c)>>16, z_b, depth_slope)
					z_b += depth_increment
					x_a += a_to_b
					x_c += c_to_a
					y_b += Rasterizer2D.width
				}
				return
			}
			y_a -= y_c
			y_c -= y_b
			for y_b = scanOffsets[y_b]; --y_c >= 0; y_b = Rasterizer2D.width {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_b, k1, uint32(x_b)>>16, uint32(x_a)>>16, z_b, depth_slope)
				z_b += depth_increment
				x_a += a_to_b
				x_b += b_to_c
			}
			for --y_a >= 0 {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_b, k1, uint32(x_c)>>16, uint32(x_a)>>16, z_b, depth_slope)
				z_b += depth_increment
				x_a += a_to_b
				x_c += c_to_a
				y_b += Rasterizer2D.width
			}
			return
		}
		x_b = x_b << 16
		x_c = x_b
		if y_b < 0 {
			x_c -= a_to_b * y_b
			x_b -= b_to_c * y_b
			z_b -= depth_increment * y_b
			y_b = 0
		}
		x_a = x_a << 16
		if y_a < 0 {
			x_a -= c_to_a * y_a
			y_a = 0
		}
		if a_to_b < b_to_c {
			y_c -= y_a
			y_a -= y_b
			for y_b = scanOffsets[y_b]; --y_a >= 0; y_b = Rasterizer2D.width {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_b, k1, uint32(x_c)>>16, uint32(x_b)>>16, z_b, depth_slope)
				z_b += depth_increment
				x_c += a_to_b
				x_b += b_to_c
			}
			for --y_c >= 0 {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_b, k1, uint32(x_a)>>16, uint32(x_b)>>16, z_b, depth_slope)
				z_b += depth_increment
				x_a += c_to_a
				x_b += b_to_c
				y_b += Rasterizer2D.width
			}
			return
		}
		y_c -= y_a
		y_a -= y_b
		for y_b = scanOffsets[y_b]; --y_a >= 0; y_b = Rasterizer2D.width {
			drawFlatTexturedScanline(Rasterizer2D.pixels, y_b, k1, uint32(x_b)>>16, uint32(x_c)>>16, z_b, depth_slope)
			z_b += depth_increment
			x_c += a_to_b
			x_b += b_to_c
		}
		for --y_c >= 0 {
			drawFlatTexturedScanline(Rasterizer2D.pixels, y_b, k1, uint32(x_b)>>16, uint32(x_a)>>16, z_b, depth_slope)
			z_b += depth_increment
			x_a += c_to_a
			x_b += b_to_c
			y_b += Rasterizer2D.width
		}
		return
	}
	if y_c >= Rasterizer2D.bottomY {
		return
	}
	if y_a > Rasterizer2D.bottomY {
		y_a = Rasterizer2D.bottomY
	}
	if y_b > Rasterizer2D.bottomY {
		y_b = Rasterizer2D.bottomY
	}
	z_c = z_c - depth_slope*x_c + depth_slope
	if y_a < y_b {
		x_c = x_c << 16
		x_b = x_c
		if y_c < 0 {
			x_b -= b_to_c * y_c
			x_c -= c_to_a * y_c
			z_c -= depth_increment * y_c
			y_c = 0
		}
		x_a = x_a << 16
		if y_a < 0 {
			x_a -= a_to_b * y_a
			y_a = 0
		}
		if b_to_c < c_to_a {
			y_b -= y_a
			y_a -= y_c
			for y_c = scanOffsets[y_c]; --y_a >= 0; y_c = Rasterizer2D.width {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_c, k1, uint32(x_b)>>16, uint32(x_c)>>16, z_c, depth_slope)
				z_c += depth_increment
				x_b += b_to_c
				x_c += c_to_a
			}
			for --y_b >= 0 {
				drawFlatTexturedScanline(Rasterizer2D.pixels, y_c, k1, uint32(x_b)>>16, uint32(x_a)>>16, z_c, depth_slope)
				z_c += depth_increment
				x_b += b_to_c
				x_a += a_to_b
				y_c += Rasterizer2D.width
			}
			return
		}
		y_b -= y_a
		y_a -= y_c
		for y_c = scanOffsets[y_c]; --y_a >= 0; y_c = Rasterizer2D.width {
			drawFlatTexturedScanline(Rasterizer2D.pixels, y_c, k1, uint32(x_c)>>16, uint32(x_b)>>16, z_c, depth_slope)
			z_c += depth_increment
			x_b += b_to_c
			x_c += c_to_a
		}
		for --y_b >= 0 {
			drawFlatTexturedScanline(Rasterizer2D.pixels, y_c, k1, uint32(x_a)>>16, uint32(x_b)>>16, z_c, depth_slope)
			z_c += depth_increment
			x_b += b_to_c
			x_a += a_to_b
			y_c += Rasterizer2D.width
		}
		return
	}
	x_c = x_c << 16
	x_a = x_c
	if y_c < 0 {
		x_a -= b_to_c * y_c
		x_c -= c_to_a * y_c
		z_c -= depth_increment * y_c
		y_c = 0
	}
	x_b = x_b << 16
	if y_b < 0 {
		x_b -= a_to_b * y_b
		y_b = 0
	}
	if b_to_c < c_to_a {
		y_a -= y_b
		y_b -= y_c
		for y_c = scanOffsets[y_c]; --y_b >= 0; y_c = Rasterizer2D.width {
			drawFlatTexturedScanline(Rasterizer2D.pixels, y_c, k1, uint32(x_a)>>16, uint32(x_c)>>16, z_c, depth_slope)
			z_c += depth_increment
			x_a += b_to_c
			x_c += c_to_a
		}
		for --y_a >= 0 {
			drawFlatTexturedScanline(Rasterizer2D.pixels, y_c, k1, uint32(x_b)>>16, uint32(x_c)>>16, z_c, depth_slope)
			z_c += depth_increment
			x_b += a_to_b
			x_c += c_to_a
			y_c += Rasterizer2D.width
		}
		return
	}
	y_a -= y_b
	y_b -= y_c
	for y_c = scanOffsets[y_c]; --y_b >= 0; y_c = Rasterizer2D.width {
		drawFlatTexturedScanline(Rasterizer2D.pixels, y_c, k1, uint32(x_c)>>16, uint32(x_a)>>16, z_c, depth_slope)
		z_c += depth_increment
		x_a += b_to_c
		x_c += c_to_a
	}
	for --y_a >= 0 {
		drawFlatTexturedScanline(Rasterizer2D.pixels, y_c, k1, uint32(x_c)>>16, uint32(x_b)>>16, z_c, depth_slope)
		z_c += depth_increment
		x_b += a_to_b
		x_c += c_to_a
		y_c += Rasterizer2D.width
	}
}
func drawShadedScanline(dest []int, offset int, x1 int, x2 int, hsl1 int, hsl2 int, depth float32, depth_slope float32) {
	var j int
	var k int
	if true {
		var l1 int
		if TextureOutOfDrawingBounds {
			if x2-x1 > 3 {
				l1 = (hsl2 - hsl1) / (x2 - x1)
			} else {
				l1 = 0
			}
			if x2 > Rasterizer2D.lastX {
				x2 = Rasterizer2D.lastX
			}
			if x1 < 0 {
				hsl1 -= x1 * l1
				x1 = 0
			}
			if x1 >= x2 {
				return
			}
			offset += x1
			k = uint32(x2-x1) >> 2
			l1 = l1 << 2
		} else {
			if x1 >= x2 {
				return
			}
			offset += x1
			k = uint32(x2-x1) >> 2
			if k > 0 {
				l1 = uint32((hsl2-hsl1)*shadowDecay[k]) >> 15
			} else {
				l1 = 0
			}
		}
		if Alpha == 0 {
			for --k >= 0 {
				j = HslToRgb[uint32(hsl1)>>8]
				hsl1 += l1
				dest[offset] = j
				offset++
				dest[offset] = j
				offset++
				dest[offset] = j
				offset++
				dest[offset] = j
				offset++
			}
			k = (x2 - x1) & 3
			if k > 0 {
				j = HslToRgb[uint32(hsl1)>>8]
				for {
					dest[offset] = j
					offset++
					if !(--k > 0) {
						break
					}
				}
				return
			}
		} else {
			a1 := Alpha
			a2 := 256 - Alpha
			for --k >= 0 {
				j = HslToRgb[uint32(hsl1)>>8]
				hsl1 += l1
				j = uint32(j&0xff00ff*a2)>>8&0xff00ff + uint32(j&0xff00*a2)>>8&0xff00
				dest[offset] = j + uint32(dest[offset]&0xff00ff*a1)>>8&0xff00ff + uint32(dest[offset]&0xff00*a1)>>8&0xff00
				offset++
				dest[offset] = j + uint32(dest[offset]&0xff00ff*a1)>>8&0xff00ff + uint32(dest[offset]&0xff00*a1)>>8&0xff00
				offset++
				dest[offset] = j + uint32(dest[offset]&0xff00ff*a1)>>8&0xff00ff + uint32(dest[offset]&0xff00*a1)>>8&0xff00
				offset++
				dest[offset] = j + uint32(dest[offset]&0xff00ff*a1)>>8&0xff00ff + uint32(dest[offset]&0xff00*a1)>>8&0xff00
				offset++
			}
			k = (x2 - x1) & 3
			if k > 0 {
				j = HslToRgb[uint32(hsl1)>>8]
				j = uint32(j&0xff00ff*a2)>>8&0xff00ff + uint32(j&0xff00*a2)>>8&0xff00
				for {
					dest[offset] = j + uint32(dest[offset]&0xff00ff*a1)>>8&0xff00ff + uint32(dest[offset]&0xff00*a1)>>8&0xff00
					offset++
					if !(--k > 0) {
						break
					}
				}
			}
		}
		return
	}
}
func DrawShadedTriangle(y1 int, y2 int, y3 int, x1 int, x2 int, x3 int, hsl1 int, hsl2 int, hsl3 int, z_a float32, z_b float32, z_c float32) {
	if z_a < 0 || z_b < 0 || z_c < 0 {
		return
	}
	j2 := 0
	k2 := 0
	if y2 != y1 {
		j2 = (x2 - x1) << 16 / (y2 - y1)
		k2 = (hsl2 - hsl1) << 15 / (y2 - y1)
	}
	l2 := 0
	i3 := 0
	if y3 != y2 {
		l2 = (x3 - x2) << 16 / (y3 - y2)
		i3 = (hsl3 - hsl2) << 15 / (y3 - y2)
	}
	j3 := 0
	k3 := 0
	if y3 != y1 {
		j3 = (x1 - x3) << 16 / (y1 - y3)
		k3 = (hsl1 - hsl3) << 15 / (y1 - y3)
	}
	b_aX := x2 - x1
	b_aY := y2 - y1
	c_aX := x3 - x1
	c_aY := y3 - y1
	b_aZ := z_b - z_a
	c_aZ := z_c - z_a
	div := b_aX*c_aY - c_aX*b_aY
	depth_slope := (b_aZ*c_aY - c_aZ*b_aY) / div
	depth_increment := (c_aZ*b_aX - b_aZ*c_aX) / div
	if y1 <= y2 && y1 <= y3 {
		if y1 >= Rasterizer2D.bottomY {
			return
		}
		if y2 > Rasterizer2D.bottomY {
			y2 = Rasterizer2D.bottomY
		}
		if y3 > Rasterizer2D.bottomY {
			y3 = Rasterizer2D.bottomY
		}
		z_a = z_a - depth_slope*x1 + depth_slope
		if y2 < y3 {
			x1 = x1 << 16
			x3 = x1
			hsl1 = hsl1 << 15
			hsl3 = hsl1
			if y1 < 0 {
				x3 -= j3 * y1
				x1 -= j2 * y1
				hsl3 -= k3 * y1
				hsl1 -= k2 * y1
				z_a -= depth_increment * y1
				y1 = 0
			}
			x2 = x2 << 16
			hsl2 = hsl2 << 15
			if y2 < 0 {
				x2 -= l2 * y2
				hsl2 -= i3 * y2
				y2 = 0
			}
			if y1 != y2 && j3 < j2 || y1 == y2 && j3 > l2 {
				y3 -= y2
				y2 -= y1
				for y1 = scanOffsets[y1]; --y2 >= 0; y1 = Rasterizer2D.width {
					Rasterizer3D.drawShadedScanline(Rasterizer2D.pixels, y1, uint32(x3)>>16, uint32(x1)>>16, uint32(hsl3)>>7, uint32(hsl1)>>7, z_a, depth_slope)
					x3 += j3
					x1 += j2
					hsl3 += k3
					hsl1 += k2
					z_a += depth_increment
				}
				for --y3 >= 0 {
					drawShadedScanline(Rasterizer2D.pixels, y1, uint32(x3)>>16, uint32(x2)>>16, uint32(hsl3)>>7, uint32(hsl2)>>7, z_a, depth_slope)
					x3 += j3
					x2 += l2
					hsl3 += k3
					hsl2 += i3
					y1 += Rasterizer2D.width
					z_a += depth_increment
				}
				return
			}
			y3 -= y2
			y2 -= y1
			for y1 = scanOffsets[y1]; --y2 >= 0; y1 = Rasterizer2D.width {
				drawShadedScanline(Rasterizer2D.pixels, y1, uint32(x1)>>16, uint32(x3)>>16, uint32(hsl1)>>7, uint32(hsl3)>>7, z_a, depth_slope)
				x3 += j3
				x1 += j2
				hsl3 += k3
				hsl1 += k2
				z_a += depth_increment
			}
			for --y3 >= 0 {
				drawShadedScanline(Rasterizer2D.pixels, y1, uint32(x2)>>16, uint32(x3)>>16, uint32(hsl2)>>7, uint32(hsl3)>>7, z_a, depth_slope)
				x3 += j3
				x2 += l2
				hsl3 += k3
				hsl2 += i3
				y1 += Rasterizer2D.width
				z_a += depth_increment
			}
			return
		}
		x1 = x1 << 16
		x2 = x1
		hsl1 = hsl1 << 15
		hsl2 = hsl1
		if y1 < 0 {
			x2 -= j3 * y1
			x1 -= j2 * y1
			hsl2 -= k3 * y1
			hsl1 -= k2 * y1
			z_a -= depth_increment * y1
			y1 = 0
		}
		x3 = x3 << 16
		hsl3 = hsl3 << 15
		if y3 < 0 {
			x3 -= l2 * y3
			hsl3 -= i3 * y3
			y3 = 0
		}
		if y1 != y3 && j3 < j2 || y1 == y3 && l2 > j2 {
			y2 -= y3
			y3 -= y1
			for y1 = scanOffsets[y1]; --y3 >= 0; y1 = Rasterizer2D.width {
				drawShadedScanline(Rasterizer2D.pixels, y1, uint32(x2)>>16, uint32(x1)>>16, uint32(hsl2)>>7, uint32(hsl1)>>7, z_a, depth_slope)
				x2 += j3
				x1 += j2
				hsl2 += k3
				hsl1 += k2
				z_a += depth_increment
			}
			for --y2 >= 0 {
				drawShadedScanline(Rasterizer2D.pixels, y1, uint32(x3)>>16, uint32(x1)>>16, uint32(hsl3)>>7, uint32(hsl1)>>7, z_a, depth_slope)
				x3 += l2
				x1 += j2
				hsl3 += i3
				hsl1 += k2
				y1 += Rasterizer2D.width
				z_a += depth_increment
			}
			return
		}
		y2 -= y3
		y3 -= y1
		for y1 = scanOffsets[y1]; --y3 >= 0; y1 = Rasterizer2D.width {
			drawShadedScanline(Rasterizer2D.pixels, y1, uint32(x1)>>16, uint32(x2)>>16, uint32(hsl1)>>7, uint32(hsl2)>>7, z_a, depth_slope)
			x2 += j3
			x1 += j2
			hsl2 += k3
			hsl1 += k2
			z_a += depth_increment
		}
		for --y2 >= 0 {
			drawShadedScanline(Rasterizer2D.pixels, y1, uint32(x1)>>16, uint32(x3)>>16, uint32(hsl1)>>7, uint32(hsl3)>>7, z_a, depth_slope)
			x3 += l2
			x1 += j2
			hsl3 += i3
			hsl1 += k2
			y1 += Rasterizer2D.width
			z_a += depth_increment
		}
		return
	}
	if y2 <= y3 {
		if y2 >= Rasterizer2D.bottomY {
			return
		}
		if y3 > Rasterizer2D.bottomY {
			y3 = Rasterizer2D.bottomY
		}
		if y1 > Rasterizer2D.bottomY {
			y1 = Rasterizer2D.bottomY
		}
		z_b = z_b - depth_slope*x2 + depth_slope
		if y3 < y1 {
			x2 = x2 << 16
			x1 = x2
			hsl2 = hsl2 << 15
			hsl1 = hsl2
			if y2 < 0 {
				x1 -= j2 * y2
				x2 -= l2 * y2
				hsl1 -= k2 * y2
				hsl2 -= i3 * y2
				z_b -= depth_increment * y2
				y2 = 0
			}
			x3 = x3 << 16
			hsl3 = hsl3 << 15
			if y3 < 0 {
				x3 -= j3 * y3
				hsl3 -= k3 * y3
				y3 = 0
			}
			if y2 != y3 && j2 < l2 || y2 == y3 && j2 > j3 {
				y1 -= y3
				y3 -= y2
				for y2 = scanOffsets[y2]; --y3 >= 0; y2 = Rasterizer2D.width {
					drawShadedScanline(Rasterizer2D.pixels, y2, uint32(x1)>>16, uint32(x2)>>16, uint32(hsl1)>>7, uint32(hsl2)>>7, z_b, depth_slope)
					x1 += j2
					x2 += l2
					hsl1 += k2
					hsl2 += i3
					z_b += depth_increment
				}
				for --y1 >= 0 {
					drawShadedScanline(Rasterizer2D.pixels, y2, uint32(x1)>>16, uint32(x3)>>16, uint32(hsl1)>>7, uint32(hsl3)>>7, z_b, depth_slope)
					x1 += j2
					x3 += j3
					hsl1 += k2
					hsl3 += k3
					y2 += Rasterizer2D.width
					z_b += depth_increment
				}
				return
			}
			y1 -= y3
			y3 -= y2
			for y2 = scanOffsets[y2]; --y3 >= 0; y2 = Rasterizer2D.width {
				drawShadedScanline(Rasterizer2D.pixels, y2, uint32(x2)>>16, uint32(x1)>>16, uint32(hsl2)>>7, uint32(hsl1)>>7, z_b, depth_slope)
				x1 += j2
				x2 += l2
				hsl1 += k2
				hsl2 += i3
				z_b += depth_increment
			}
			for --y1 >= 0 {
				drawShadedScanline(Rasterizer2D.pixels, y2, uint32(x3)>>16, uint32(x1)>>16, uint32(hsl3)>>7, uint32(hsl1)>>7, z_b, depth_slope)
				x1 += j2
				x3 += j3
				hsl1 += k2
				hsl3 += k3
				y2 += Rasterizer2D.width
				z_b += depth_increment
			}
			return
		}
		x2 = x2 << 16
		x3 = x2
		hsl2 = hsl2 << 15
		hsl3 = hsl2
		if y2 < 0 {
			x3 -= j2 * y2
			x2 -= l2 * y2
			hsl3 -= k2 * y2
			hsl2 -= i3 * y2
			z_b -= depth_increment * y2
			y2 = 0
		}
		x1 = x1 << 16
		hsl1 = hsl1 << 15
		if y1 < 0 {
			x1 -= j3 * y1
			hsl1 -= k3 * y1
			y1 = 0
		}
		if j2 < l2 {
			y3 -= y1
			y1 -= y2
			for y2 = scanOffsets[y2]; --y1 >= 0; y2 = Rasterizer2D.width {
				drawShadedScanline(Rasterizer2D.pixels, y2, uint32(x3)>>16, uint32(x2)>>16, uint32(hsl3)>>7, uint32(hsl2)>>7, z_b, depth_slope)
				x3 += j2
				x2 += l2
				hsl3 += k2
				hsl2 += i3
				z_b += depth_increment
			}
			for --y3 >= 0 {
				drawShadedScanline(Rasterizer2D.pixels, y2, uint32(x1)>>16, uint32(x2)>>16, uint32(hsl1)>>7, uint32(hsl2)>>7, z_b, depth_slope)
				x1 += j3
				x2 += l2
				hsl1 += k3
				hsl2 += i3
				y2 += Rasterizer2D.width
				z_b += depth_increment
			}
			return
		}
		y3 -= y1
		y1 -= y2
		for y2 = scanOffsets[y2]; --y1 >= 0; y2 = Rasterizer2D.width {
			drawShadedScanline(Rasterizer2D.pixels, y2, uint32(x2)>>16, uint32(x3)>>16, uint32(hsl2)>>7, uint32(hsl3)>>7, z_b, depth_slope)
			x3 += j2
			x2 += l2
			hsl3 += k2
			hsl2 += i3
			z_b += depth_increment
		}
		for --y3 >= 0 {
			drawShadedScanline(Rasterizer2D.pixels, y2, uint32(x2)>>16, uint32(x1)>>16, uint32(hsl2)>>7, uint32(hsl1)>>7, z_b, depth_slope)
			x1 += j3
			x2 += l2
			hsl1 += k3
			hsl2 += i3
			y2 += Rasterizer2D.width
			z_b += depth_increment
		}
		return
	}
	if y3 >= Rasterizer2D.bottomY {
		return
	}
	if y1 > Rasterizer2D.bottomY {
		y1 = Rasterizer2D.bottomY
	}
	if y2 > Rasterizer2D.bottomY {
		y2 = Rasterizer2D.bottomY
	}
	z_c = z_c - depth_slope*x3 + depth_slope
	if y1 < y2 {
		x3 = x3 << 16
		x2 = x3
		hsl3 = hsl3 << 15
		hsl2 = hsl3
		if y3 < 0 {
			x2 -= l2 * y3
			x3 -= j3 * y3
			hsl2 -= i3 * y3
			hsl3 -= k3 * y3
			z_c -= depth_increment * y3
			y3 = 0
		}
		x1 = x1 << 16
		hsl1 = hsl1 << 15
		if y1 < 0 {
			x1 -= j2 * y1
			hsl1 -= k2 * y1
			y1 = 0
		}
		if l2 < j3 {
			y2 -= y1
			y1 -= y3
			for y3 = scanOffsets[y3]; --y1 >= 0; y3 = Rasterizer2D.width {
				drawShadedScanline(Rasterizer2D.pixels, y3, uint32(x2)>>16, uint32(x3)>>16, uint32(hsl2)>>7, uint32(hsl3)>>7, z_c, depth_slope)
				x2 += l2
				x3 += j3
				hsl2 += i3
				hsl3 += k3
				z_c += depth_increment
			}
			for --y2 >= 0 {
				drawShadedScanline(Rasterizer2D.pixels, y3, uint32(x2)>>16, uint32(x1)>>16, uint32(hsl2)>>7, uint32(hsl1)>>7, z_c, depth_slope)
				x2 += l2
				x1 += j2
				hsl2 += i3
				hsl1 += k2
				y3 += Rasterizer2D.width
				z_c += depth_increment
			}
			return
		}
		y2 -= y1
		y1 -= y3
		for y3 = scanOffsets[y3]; --y1 >= 0; y3 = Rasterizer2D.width {
			drawShadedScanline(Rasterizer2D.pixels, y3, uint32(x3)>>16, uint32(x2)>>16, uint32(hsl3)>>7, uint32(hsl2)>>7, z_c, depth_slope)
			x2 += l2
			x3 += j3
			hsl2 += i3
			hsl3 += k3
			z_c += depth_increment
		}
		for --y2 >= 0 {
			drawShadedScanline(Rasterizer2D.pixels, y3, uint32(x1)>>16, uint32(x2)>>16, uint32(hsl1)>>7, uint32(hsl2)>>7, z_c, depth_slope)
			x2 += l2
			x1 += j2
			hsl2 += i3
			hsl1 += k2
			y3 += Rasterizer2D.width
			z_c += depth_increment
		}
		return
	}
	x3 = x3 << 16
	x1 = x3
	hsl3 = hsl3 << 15
	hsl1 = hsl3
	if y3 < 0 {
		x1 -= l2 * y3
		x3 -= j3 * y3
		hsl1 -= i3 * y3
		hsl3 -= k3 * y3
		z_c -= depth_increment * y3
		y3 = 0
	}
	x2 = x2 << 16
	hsl2 = hsl2 << 15
	if y2 < 0 {
		x2 -= j2 * y2
		hsl2 -= k2 * y2
		y2 = 0
	}
	if l2 < j3 {
		y1 -= y2
		y2 -= y3
		for y3 = scanOffsets[y3]; --y2 >= 0; y3 = Rasterizer2D.width {
			drawShadedScanline(Rasterizer2D.pixels, y3, uint32(x1)>>16, uint32(x3)>>16, uint32(hsl1)>>7, uint32(hsl3)>>7, z_c, depth_slope)
			x1 += l2
			x3 += j3
			hsl1 += i3
			hsl3 += k3
			z_c += depth_increment
		}
		for --y1 >= 0 {
			drawShadedScanline(Rasterizer2D.pixels, y3, uint32(x2)>>16, uint32(x3)>>16, uint32(hsl2)>>7, uint32(hsl3)>>7, z_c, depth_slope)
			x2 += j2
			x3 += j3
			hsl2 += k2
			hsl3 += k3
			y3 += Rasterizer2D.width
			z_c += depth_increment
		}
		return
	}
	y1 -= y2
	y2 -= y3
	for y3 = scanOffsets[y3]; --y2 >= 0; y3 = Rasterizer2D.width {
		drawShadedScanline(Rasterizer2D.pixels, y3, uint32(x3)>>16, uint32(x1)>>16, uint32(hsl3)>>7, uint32(hsl1)>>7, z_c, depth_slope)
		x1 += l2
		x3 += j3
		hsl1 += i3
		hsl3 += k3
		z_c += depth_increment
	}
	for --y1 >= 0 {
		drawShadedScanline(Rasterizer2D.pixels, y3, uint32(x3)>>16, uint32(x2)>>16, uint32(hsl3)>>7, uint32(hsl2)>>7, z_c, depth_slope)
		x2 += j2
		x3 += j3
		hsl2 += k2
		hsl3 += k3
		y3 += Rasterizer2D.width
		z_c += depth_increment
	}
}
func DrawTexturedScanline(dest []int, texture []int, dest_off int, start_x int, end_x int, shadeValue int, gradient int, l1 int, i2 int, j2 int, k2 int, l2 int, i3 int, depth float32, depth_slope float32) {
	rgb := 0
	loops := 0
	if start_x >= end_x {
		return
	}
	var j3 int
	var k3 int
	if TextureOutOfDrawingBounds {
		j3 = (gradient - shadeValue) / (end_x - start_x)
		if end_x > Rasterizer2D.lastX {
			end_x = Rasterizer2D.lastX
		}
		if start_x < 0 {
			shadeValue -= start_x * j3
			start_x = 0
		}
		if start_x >= end_x {
			return
		}
		k3 = uint32(end_x-start_x) >> 3
		j3 = j3 << 12
		shadeValue = shadeValue << 9
	} else {
		if end_x-start_x > 7 {
			k3 = uint32(end_x-start_x) >> 3
			j3 = uint32((gradient-shadeValue)*shadowDecay[k3]) >> 6
		} else {
			k3 = 0
			j3 = 0
		}
		shadeValue = shadeValue << 9
	}
	dest_off += start_x
	depth += depth_slope * start_x
	j4 := 0
	l4 := 0
	l6 := start_x - OriginViewX
	l1 += uint32(k2) >> 3 * l6
	i2 += uint32(l2) >> 3 * l6
	j2 += uint32(i3) >> 3 * l6
	l5 := uint32(j2) >> 14
	if l5 != 0 {
		rgb = l1 / l5
		loops = i2 / l5
		if rgb < 0 {
			rgb = 0
		} else if rgb > 16256 {
			rgb = 16256
		}
	}
	l1 += k2
	i2 += l2
	j2 += i3
	l5 = uint32(j2) >> 14
	if l5 != 0 {
		j4 = l1 / l5
		l4 = i2 / l5
		if j4 < 7 {
			j4 = 7
		} else if j4 > 16256 {
			j4 = 16256
		}
	}
	j7 := uint32(j4-rgb) >> 3
	l7 := uint32(l4-loops) >> 3
	rgb += shadeValue & 0x600000
	j8 := uint32(shadeValue) >> 23
	if textureIsNotTransparant {
		for --k3 > 0 {
			for i := 0; i < 8; i++ {
				if true {
					dest[dest_off] = uint32(texture[loops&0x3f80+uint32(rgb)>>7]) >> j8
					<<unimp_arrayref_Rasterizer2D.depthBuffer>>[dest_off] = depth
				}
				depth += depth_slope
				dest_off++
				rgb += j7
				loops += l7
			}
			rgb = j4
			loops = l4
			l1 += k2
			i2 += l2
			j2 += i3
			i6 := uint32(j2) >> 14
			if i6 != 0 {
				j4 = l1 / i6
				l4 = i2 / i6
				if j4 < 7 {
					j4 = 7
				} else if j4 > 16256 {
					j4 = 16256
				}
			}
			j7 = uint32(j4-rgb) >> 3
			l7 = uint32(l4-loops) >> 3
			shadeValue += j3
			rgb += shadeValue & 0x600000
			j8 = uint32(shadeValue) >> 23
		}
		for k3 = (end_x-start_x)&7; --k3 > 0; {
			if true {
				dest[dest_off] = uint32(texture[loops&0x3f80+uint32(rgb)>>7]) >> j8
				<<unimp_arrayref_Rasterizer2D.depthBuffer>>[dest_off] = depth
			}
			dest_off++
			depth += depth_slope
			rgb += j7
			loops += l7
		}
		return
	}
	for --k3 > 0 {
		var i9 int
		for i := 0; i < 8; i++ {
			if (i9 = uint32(texture[loops&0x3f80+uint32(rgb)>>7])>>j8) != 0 {
				dest[dest_off] = i9
				<<unimp_arrayref_Rasterizer2D.depthBuffer>>[dest_off] = depth
			}
			dest_off++
			depth += depth_slope
			rgb += j7
			loops += l7
		}
		rgb = j4
		loops = l4
		l1 += k2
		i2 += l2
		j2 += i3
		j6 := uint32(j2) >> 14
		if j6 != 0 {
			j4 = l1 / j6
			l4 = i2 / j6
			if j4 < 7 {
				j4 = 7
			} else if j4 > 16256 {
				j4 = 16256
			}
		}
		j7 = uint32(j4-rgb) >> 3
		l7 = uint32(l4-loops) >> 3
		shadeValue += j3
		rgb += shadeValue & 0x600000
		j8 = uint32(shadeValue) >> 23
	}
	for l3 := (end_x - start_x) & 7; --l3 > 0; {
		var j9 int
		if (j9 = uint32(texture[loops&0x3f80+uint32(rgb)>>7])>>j8) != 0 {
			dest[dest_off] = j9
			<<unimp_arrayref_Rasterizer2D.depthBuffer>>[dest_off] = depth
		}
		depth += depth_slope
		dest_off++
		rgb += j7
		loops += l7
	}
}
func DrawTexturedTriangle(y_a int, y_b int, y_c int, x_a int, x_b int, x_c int, k1 int, l1 int, i2 int, Px int, Mx int, Nx int, Pz int, Mz int, Nz int, Py int, My int, Ny int, k4 int, z_a float32, z_b float32, z_c float32) {
	if z_a < 0 || z_b < 0 || z_c < 0 {
		return
	}
	texture := Rasterizer3D.getTexturePixels(k4)
	textureIsNotTransparant = !textureIsTransparant[k4]
	Mx = Px - Mx
	Mz = Pz - Mz
	My = Py - My
	Nx -= Px
	Nz -= Pz
	Ny -= Py
	Oa := (Nx*Pz - Nz*Px) << <<unimp_expr[*grammar.JConditionalExpr]>>
	Ha := (Nz*Py - Ny*Pz) << 8
	Va := (Ny*Px - Nx*Py) << 5
	Ob := (Mx*Pz - Mz*Px) << <<unimp_expr[*grammar.JConditionalExpr]>>
	Hb := (Mz*Py - My*Pz) << 8
	Vb := (My*Px - Mx*Py) << 5
	Oc := (Mz*Nx - Mx*Nz) << <<unimp_expr[*grammar.JConditionalExpr]>>
	Hc := (My*Nz - Mz*Ny) << 8
	Vc := (Mx*Ny - My*Nx) << 5
	a_to_b := 0
	grad_a_off := 0
	if y_b != y_a {
		a_to_b = (x_b - x_a) << 16 / (y_b - y_a)
		grad_a_off = (l1 - k1) << 16 / (y_b - y_a)
	}
	b_to_c := 0
	grad_b_off := 0
	if y_c != y_b {
		b_to_c = (x_c - x_b) << 16 / (y_c - y_b)
		grad_b_off = (i2 - l1) << 16 / (y_c - y_b)
	}
	c_to_a := 0
	grad_c_off := 0
	if y_c != y_a {
		c_to_a = (x_a - x_c) << 16 / (y_a - y_c)
		grad_c_off = (k1 - i2) << 16 / (y_a - y_c)
	}
	b_aX := x_b - x_a
	b_aY := y_b - y_a
	c_aX := x_c - x_a
	c_aY := y_c - y_a
	b_aZ := z_b - z_a
	c_aZ := z_c - z_a
	div := b_aX*c_aY - c_aX*b_aY
	depth_slope := (b_aZ*c_aY - c_aZ*b_aY) / div
	depth_increment := (c_aZ*b_aX - b_aZ*c_aX) / div
	if y_a <= y_b && y_a <= y_c {
		if y_a >= Rasterizer2D.bottomY {
			return
		}
		if y_b > Rasterizer2D.bottomY {
			y_b = Rasterizer2D.bottomY
		}
		if y_c > Rasterizer2D.bottomY {
			y_c = Rasterizer2D.bottomY
		}
		z_a = z_a - depth_slope*x_a + depth_slope
		if y_b < y_c {
			x_a = x_a << 16
			x_c = x_a
			k1 = k1 << 16
			i2 = k1
			if y_a < 0 {
				x_c -= c_to_a * y_a
				x_a -= a_to_b * y_a
				z_a -= depth_increment * y_a
				i2 -= grad_c_off * y_a
				k1 -= grad_a_off * y_a
				y_a = 0
			}
			x_b = x_b << 16
			l1 = l1 << 16
			if y_b < 0 {
				x_b -= b_to_c * y_b
				l1 -= grad_b_off * y_b
				y_b = 0
			}
			k8 := y_a - OriginViewY
			Oa += Va * k8
			Ob += Vb * k8
			Oc += Vc * k8
			if y_a != y_b && c_to_a < a_to_b || y_a == y_b && c_to_a > b_to_c {
				y_c -= y_b
				y_b -= y_a
				y_a = scanOffsets[y_a]
				for --y_b >= 0 {
					Rasterizer3D.DrawTexturedScanline(Rasterizer2D.pixels, texture, y_a, uint32(x_c)>>16, uint32(x_a)>>16, uint32(i2)>>8, uint32(k1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_a, depth_slope)
					x_c += c_to_a
					x_a += a_to_b
					z_a += depth_increment
					i2 += grad_c_off
					k1 += grad_a_off
					y_a += Rasterizer2D.width
					Oa += Va
					Ob += Vb
					Oc += Vc
				}
				for --y_c >= 0 {
					drawTexturedScanline(Rasterizer2D.pixels, texture, y_a, uint32(x_c)>>16, uint32(x_b)>>16, uint32(i2)>>8, uint32(l1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_a, depth_slope)
					x_c += c_to_a
					x_b += b_to_c
					z_a += depth_increment
					i2 += grad_c_off
					l1 += grad_b_off
					y_a += Rasterizer2D.width
					Oa += Va
					Ob += Vb
					Oc += Vc
				}
				return
			}
			y_c -= y_b
			y_b -= y_a
			y_a = scanOffsets[y_a]
			for --y_b >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_a, uint32(x_a)>>16, uint32(x_c)>>16, uint32(k1)>>8, uint32(i2)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_a, depth_slope)
				x_c += c_to_a
				x_a += a_to_b
				z_a += depth_increment
				i2 += grad_c_off
				k1 += grad_a_off
				y_a += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			for --y_c >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_a, uint32(x_b)>>16, uint32(x_c)>>16, uint32(l1)>>8, uint32(i2)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_a, depth_slope)
				x_c += c_to_a
				x_b += b_to_c
				z_a += depth_increment
				i2 += grad_c_off
				l1 += grad_b_off
				y_a += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			return
		}
		x_a = x_a << 16
		x_b = x_a
		k1 = k1 << 16
		l1 = k1
		if y_a < 0 {
			x_b -= c_to_a * y_a
			x_a -= a_to_b * y_a
			z_a -= depth_increment * y_a
			l1 -= grad_c_off * y_a
			k1 -= grad_a_off * y_a
			y_a = 0
		}
		x_c = x_c << 16
		i2 = i2 << 16
		if y_c < 0 {
			x_c -= b_to_c * y_c
			i2 -= grad_b_off * y_c
			y_c = 0
		}
		l8 := y_a - OriginViewY
		Oa += Va * l8
		Ob += Vb * l8
		Oc += Vc * l8
		if y_a != y_c && c_to_a < a_to_b || y_a == y_c && b_to_c > a_to_b {
			y_b -= y_c
			y_c -= y_a
			y_a = scanOffsets[y_a]
			for --y_c >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_a, uint32(x_b)>>16, uint32(x_a)>>16, uint32(l1)>>8, uint32(k1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_a, depth_slope)
				x_b += c_to_a
				x_a += a_to_b
				l1 += grad_c_off
				k1 += grad_a_off
				z_a += depth_increment
				y_a += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			for --y_b >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_a, uint32(x_c)>>16, uint32(x_a)>>16, uint32(i2)>>8, uint32(k1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_a, depth_slope)
				x_c += b_to_c
				x_a += a_to_b
				i2 += grad_b_off
				k1 += grad_a_off
				z_a += depth_increment
				y_a += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			return
		}
		y_b -= y_c
		y_c -= y_a
		y_a = scanOffsets[y_a]
		for --y_c >= 0 {
			drawTexturedScanline(Rasterizer2D.pixels, texture, y_a, uint32(x_a)>>16, uint32(x_b)>>16, uint32(k1)>>8, uint32(l1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_a, depth_slope)
			x_b += c_to_a
			x_a += a_to_b
			l1 += grad_c_off
			k1 += grad_a_off
			z_a += depth_increment
			y_a += Rasterizer2D.width
			Oa += Va
			Ob += Vb
			Oc += Vc
		}
		for --y_b >= 0 {
			drawTexturedScanline(Rasterizer2D.pixels, texture, y_a, uint32(x_a)>>16, uint32(x_c)>>16, uint32(k1)>>8, uint32(i2)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_a, depth_slope)
			x_c += b_to_c
			x_a += a_to_b
			i2 += grad_b_off
			k1 += grad_a_off
			z_a += depth_increment
			y_a += Rasterizer2D.width
			Oa += Va
			Ob += Vb
			Oc += Vc
		}
		return
	}
	if y_b <= y_c {
		if y_b >= Rasterizer2D.bottomY {
			return
		}
		if y_c > Rasterizer2D.bottomY {
			y_c = Rasterizer2D.bottomY
		}
		if y_a > Rasterizer2D.bottomY {
			y_a = Rasterizer2D.bottomY
		}
		z_b = z_b - depth_slope*x_b + depth_slope
		if y_c < y_a {
			x_b = x_b << 16
			x_a = x_b
			l1 = l1 << 16
			k1 = l1
			if y_b < 0 {
				x_a -= a_to_b * y_b
				x_b -= b_to_c * y_b
				z_b -= depth_increment * y_b
				k1 -= grad_a_off * y_b
				l1 -= grad_b_off * y_b
				y_b = 0
			}
			x_c = x_c << 16
			i2 = i2 << 16
			if y_c < 0 {
				x_c -= c_to_a * y_c
				i2 -= grad_c_off * y_c
				y_c = 0
			}
			i9 := y_b - OriginViewY
			Oa += Va * i9
			Ob += Vb * i9
			Oc += Vc * i9
			if y_b != y_c && a_to_b < b_to_c || y_b == y_c && a_to_b > c_to_a {
				y_a -= y_c
				y_c -= y_b
				y_b = scanOffsets[y_b]
				for --y_c >= 0 {
					drawTexturedScanline(Rasterizer2D.pixels, texture, y_b, uint32(x_a)>>16, uint32(x_b)>>16, uint32(k1)>>8, uint32(l1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_b, depth_slope)
					x_a += a_to_b
					x_b += b_to_c
					k1 += grad_a_off
					l1 += grad_b_off
					z_b += depth_increment
					y_b += Rasterizer2D.width
					Oa += Va
					Ob += Vb
					Oc += Vc
				}
				for --y_a >= 0 {
					drawTexturedScanline(Rasterizer2D.pixels, texture, y_b, uint32(x_a)>>16, uint32(x_c)>>16, uint32(k1)>>8, uint32(i2)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_b, depth_slope)
					x_a += a_to_b
					x_c += c_to_a
					k1 += grad_a_off
					i2 += grad_c_off
					z_b += depth_increment
					y_b += Rasterizer2D.width
					Oa += Va
					Ob += Vb
					Oc += Vc
				}
				return
			}
			y_a -= y_c
			y_c -= y_b
			y_b = scanOffsets[y_b]
			for --y_c >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_b, uint32(x_b)>>16, uint32(x_a)>>16, uint32(l1)>>8, uint32(k1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_b, depth_slope)
				x_a += a_to_b
				x_b += b_to_c
				k1 += grad_a_off
				l1 += grad_b_off
				z_b += depth_increment
				y_b += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			for --y_a >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_b, uint32(x_c)>>16, uint32(x_a)>>16, uint32(i2)>>8, uint32(k1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_b, depth_slope)
				x_a += a_to_b
				x_c += c_to_a
				k1 += grad_a_off
				i2 += grad_c_off
				z_b += depth_increment
				y_b += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			return
		}
		x_b = x_b << 16
		x_c = x_b
		l1 = l1 << 16
		i2 = l1
		if y_b < 0 {
			x_c -= a_to_b * y_b
			x_b -= b_to_c * y_b
			z_b -= depth_increment * y_b
			i2 -= grad_a_off * y_b
			l1 -= grad_b_off * y_b
			y_b = 0
		}
		x_a = x_a << 16
		k1 = k1 << 16
		if y_a < 0 {
			x_a -= c_to_a * y_a
			k1 -= grad_c_off * y_a
			y_a = 0
		}
		j9 := y_b - OriginViewY
		Oa += Va * j9
		Ob += Vb * j9
		Oc += Vc * j9
		if a_to_b < b_to_c {
			y_c -= y_a
			y_a -= y_b
			y_b = scanOffsets[y_b]
			for --y_a >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_b, uint32(x_c)>>16, uint32(x_b)>>16, uint32(i2)>>8, uint32(l1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_b, depth_slope)
				x_c += a_to_b
				x_b += b_to_c
				i2 += grad_a_off
				l1 += grad_b_off
				z_b += depth_increment
				y_b += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			for --y_c >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_b, uint32(x_a)>>16, uint32(x_b)>>16, uint32(k1)>>8, uint32(l1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_b, depth_slope)
				x_a += c_to_a
				x_b += b_to_c
				k1 += grad_c_off
				l1 += grad_b_off
				z_b += depth_increment
				y_b += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			return
		}
		y_c -= y_a
		y_a -= y_b
		y_b = scanOffsets[y_b]
		for --y_a >= 0 {
			drawTexturedScanline(Rasterizer2D.pixels, texture, y_b, uint32(x_b)>>16, uint32(x_c)>>16, uint32(l1)>>8, uint32(i2)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_b, depth_slope)
			x_c += a_to_b
			x_b += b_to_c
			i2 += grad_a_off
			l1 += grad_b_off
			z_b += depth_increment
			y_b += Rasterizer2D.width
			Oa += Va
			Ob += Vb
			Oc += Vc
		}
		for --y_c >= 0 {
			drawTexturedScanline(Rasterizer2D.pixels, texture, y_b, uint32(x_b)>>16, uint32(x_a)>>16, uint32(l1)>>8, uint32(k1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_b, depth_slope)
			x_a += c_to_a
			x_b += b_to_c
			k1 += grad_c_off
			l1 += grad_b_off
			z_b += depth_increment
			y_b += Rasterizer2D.width
			Oa += Va
			Ob += Vb
			Oc += Vc
		}
		return
	}
	if y_c >= Rasterizer2D.bottomY {
		return
	}
	if y_a > Rasterizer2D.bottomY {
		y_a = Rasterizer2D.bottomY
	}
	if y_b > Rasterizer2D.bottomY {
		y_b = Rasterizer2D.bottomY
	}
	z_c = z_c - depth_slope*x_c + depth_slope
	if y_a < y_b {
		x_c = x_c << 16
		x_b = x_c
		i2 = i2 << 16
		l1 = i2
		if y_c < 0 {
			x_b -= b_to_c * y_c
			x_c -= c_to_a * y_c
			z_c -= depth_increment * y_c
			l1 -= grad_b_off * y_c
			i2 -= grad_c_off * y_c
			y_c = 0
		}
		x_a = x_a << 16
		k1 = k1 << 16
		if y_a < 0 {
			x_a -= a_to_b * y_a
			k1 -= grad_a_off * y_a
			y_a = 0
		}
		k9 := y_c - OriginViewY
		Oa += Va * k9
		Ob += Vb * k9
		Oc += Vc * k9
		if b_to_c < c_to_a {
			y_b -= y_a
			y_a -= y_c
			y_c = scanOffsets[y_c]
			for --y_a >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_c, uint32(x_b)>>16, uint32(x_c)>>16, uint32(l1)>>8, uint32(i2)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_c, depth_slope)
				x_b += b_to_c
				x_c += c_to_a
				l1 += grad_b_off
				i2 += grad_c_off
				z_c += depth_increment
				y_c += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			for --y_b >= 0 {
				drawTexturedScanline(Rasterizer2D.pixels, texture, y_c, uint32(x_b)>>16, uint32(x_a)>>16, uint32(l1)>>8, uint32(k1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_c, depth_slope)
				x_b += b_to_c
				x_a += a_to_b
				l1 += grad_b_off
				k1 += grad_a_off
				z_c += depth_increment
				y_c += Rasterizer2D.width
				Oa += Va
				Ob += Vb
				Oc += Vc
			}
			return
		}
		y_b -= y_a
		y_a -= y_c
		y_c = scanOffsets[y_c]
		for --y_a >= 0 {
			drawTexturedScanline(Rasterizer2D.pixels, texture, y_c, uint32(x_c)>>16, uint32(x_b)>>16, uint32(i2)>>8, uint32(l1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_c, depth_slope)
			x_b += b_to_c
			x_c += c_to_a
			l1 += grad_b_off
			i2 += grad_c_off
			z_c += depth_increment
			y_c += Rasterizer2D.width
			Oa += Va
			Ob += Vb
			Oc += Vc
		}
		for --y_b >= 0 {
			drawTexturedScanline(Rasterizer2D.pixels, texture, y_c, uint32(x_a)>>16, uint32(x_b)>>16, uint32(k1)>>8, uint32(l1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_c, depth_slope)
			x_b += b_to_c
			x_a += a_to_b
			l1 += grad_b_off
			k1 += grad_a_off
			z_c += depth_increment
			y_c += Rasterizer2D.width
			Oa += Va
			Ob += Vb
			Oc += Vc
		}
		return
	}
	x_c = x_c << 16
	x_a = x_c
	i2 = i2 << 16
	k1 = i2
	if y_c < 0 {
		x_a -= b_to_c * y_c
		x_c -= c_to_a * y_c
		z_c -= depth_increment * y_c
		k1 -= grad_b_off * y_c
		i2 -= grad_c_off * y_c
		y_c = 0
	}
	x_b = x_b << 16
	l1 = l1 << 16
	if y_b < 0 {
		x_b -= a_to_b * y_b
		l1 -= grad_a_off * y_b
		y_b = 0
	}
	l9 := y_c - OriginViewY
	Oa += Va * l9
	Ob += Vb * l9
	Oc += Vc * l9
	if b_to_c < c_to_a {
		y_a -= y_b
		y_b -= y_c
		y_c = scanOffsets[y_c]
		for --y_b >= 0 {
			drawTexturedScanline(Rasterizer2D.pixels, texture, y_c, uint32(x_a)>>16, uint32(x_c)>>16, uint32(k1)>>8, uint32(i2)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_c, depth_slope)
			x_a += b_to_c
			x_c += c_to_a
			k1 += grad_b_off
			i2 += grad_c_off
			z_c += depth_increment
			y_c += Rasterizer2D.width
			Oa += Va
			Ob += Vb
			Oc += Vc
		}
		for --y_a >= 0 {
			drawTexturedScanline(Rasterizer2D.pixels, texture, y_c, uint32(x_b)>>16, uint32(x_c)>>16, uint32(l1)>>8, uint32(i2)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_c, depth_slope)
			x_b += a_to_b
			x_c += c_to_a
			l1 += grad_a_off
			i2 += grad_c_off
			z_c += depth_increment
			y_c += Rasterizer2D.width
			Oa += Va
			Ob += Vb
			Oc += Vc
		}
		return
	}
	y_a -= y_b
	y_b -= y_c
	y_c = scanOffsets[y_c]
	for --y_b >= 0 {
		drawTexturedScanline(Rasterizer2D.pixels, texture, y_c, uint32(x_c)>>16, uint32(x_a)>>16, uint32(i2)>>8, uint32(k1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_c, depth_slope)
		x_a += b_to_c
		x_c += c_to_a
		k1 += grad_b_off
		i2 += grad_c_off
		z_c += depth_increment
		y_c += Rasterizer2D.width
		Oa += Va
		Ob += Vb
		Oc += Vc
	}
	for --y_a >= 0 {
		drawTexturedScanline(Rasterizer2D.pixels, texture, y_c, uint32(x_c)>>16, uint32(x_b)>>16, uint32(i2)>>8, uint32(l1)>>8, Oa, Ob, Oc, Ha, Hb, Hc, z_c, depth_slope)
		x_b += a_to_b
		x_c += c_to_a
		l1 += grad_a_off
		i2 += grad_c_off
		z_c += depth_increment
		y_c += Rasterizer2D.width
		Oa += Va
		Ob += Vb
		Oc += Vc
	}
}
func GetOverallColour(textureId int) (int) {
	if averageTextureColours[textureId] != 0 {
		return averageTextureColours[textureId]
	}
	totalRed := 0
	totalGreen := 0
	totalBlue := 0
	colourCount := <<unimp_obj.nm_*parser.GoArrayReference>>
	for ptr := 0; ptr < colourCount; ptr++ {
		totalRed += uint32(currentPalette[textureId][ptr]) >> 16 & 0xff
		totalGreen += uint32(currentPalette[textureId][ptr]) >> 8 & 0xff
		totalBlue += currentPalette[textureId][ptr] & 0xff
	}
	avgPaletteColour := totalRed/colourCount<<16 + totalGreen/colourCount<<8 + totalBlue/colourCount
	avgPaletteColour = Rasterizer3D.adjustBrightness(avgPaletteColour, 1.3999999999999999D)
	if avgPaletteColour == 0 {
		avgPaletteColour = 1
	}
	averageTextureColours[textureId] = avgPaletteColour
	return avgPaletteColour
}
func getTexturePixels(textureId int) (<<array>>) {
	textureLastUsed[textureId] = ++lastTextureRetrievalCount
	if texturesPixelBuffer[textureId] != nil {
		return texturesPixelBuffer[textureId]
	}
	var texturePixels []int
	if textureRequestBufferPointer > 0 {
		texturePixels = textureRequestPixelBuffer[--textureRequestBufferPointer]
		textureRequestPixelBuffer[textureRequestBufferPointer] = nil
	} else {
		lastUsed := 0
		target := -1
		for l := 0; l < textureCount; l++ {
			if texturesPixelBuffer[l] != nil && (textureLastUsed[l] < lastUsed || target == -1) {
				lastUsed = textureLastUsed[l]
				target = l
			}
		}
		texturePixels = texturesPixelBuffer[target]
		texturesPixelBuffer[target] = nil
	}
	texturesPixelBuffer[textureId] = texturePixels
	background := textures[textureId]
	texturePalette := currentPalette[textureId]
	if background.width == 64 {
		for x := 0; x < 128; x++ {
			for y := 0; y < 128; y++ {
				texturePixels[y+x<<7] = texturePalette[background[uint32(y)>>1+uint32(x)>>1<<6]]
			}
		}
	} else {
		for i := 0; i < 16384; i++ {
			texturePixels[i] = texturePalette[background[i]]
		}
	}
	textureIsTransparant[textureId] = false
	for i := 0; i < 16384; i++ {
		texturePixels[i] &= 0xf8f8ff
		colour := texturePixels[i]
		if colour == 0 {
			textureIsTransparant[textureId] = true
		}
		texturePixels[16384+i] = (colour - uint32(colour)>>3) & 0xf8f8ff
		texturePixels[32768+i] = (colour - uint32(colour)>>2) & 0xf8f8ff
		texturePixels[49152+i] = (colour - uint32(colour)>>2 - uint32(colour)>>3) & 0xf8f8ff
	}
	return texturePixels
}
func init() {
	shadowDecay = make([]int, 512)
	DEPTH = make([]int, 2048)
	Sine = make([]int, 2048)
	Cosine = make([]int, 2048)
	for index := 1; index < 512; index++ {
		shadowDecay[index] = 32768 / index
	}
	for index := 1; index < 2048; index++ {
		DEPTH[index] = 0x10000 / index
	}
	for k := 0; k < 2048; k++ {
		Sine[k] = (65536D * Math.sin(k*0.0030679614999999999D)).(int)
		Cosine[k] = (65536D * Math.cos(k*0.0030679614999999999D)).(int)
	}
}
func Initialize(archive *FileArchive) {
	textureCount = 0
	for index := 0; index < tEXTURE_LENGTH; index++ {
		if try() {
			textures[index] = NewIndexedImage(archive, String.valueOf(index), 0)
			textures[index].resize()
			textureCount++
		} else if catch_Exception(ex) {
			ex.printStackTrace()
		}
	}
}
func InitiateRequestBuffers() {
	if textureRequestPixelBuffer == nil {
		textureRequestBufferPointer = 20
		textureRequestPixelBuffer = make([]int, textureRequestBufferPointer, 0x10000)
		for index := 0; index < 50; index++ {
			texturesPixelBuffer[index] = nil
		}
	}
}
func RequestTextureUpdate(textureId int) {
	if texturesPixelBuffer[textureId] == nil {
		return
	}
	textureRequestPixelBuffer[++textureRequestBufferPointer] = texturesPixelBuffer[textureId]
	texturesPixelBuffer[textureId] = nil
}
func SetBrightness(brightness float64) {
	size := 0
	for index := 0; index < 512; index++ {
		d1 := index/8/64D + 0.0078125D
		d2 := index&7/8D + 0.0625D
		for step := 0; step < 128; step++ {
			d3 := step / 128D
			r := d3
			g := d3
			b := d3
			if d2 != 0.0D {
				var d7 float64
				if d3 < 0.5D {
					d7 = d3 * (1.0D + d2)
				} else {
					d7 = d3 + d2 - d3*d2
				}
				d8 := 2D*d3 - d7
				d9 := d1 + 0.33333333333333331D
				if d9 > 1.0D {
					d9--
				}
				d10 := d1
				d11 := d1 - 0.33333333333333331D
				if d11 < 0.0D {
					d11++
				}
				if 6D*d9 < 1.0D {
					r = d8 + (d7-d8)*6D*d9
				} else if 2D*d9 < 1.0D {
					r = d7
				} else if 3D*d9 < 2D {
					r = d8 + (d7-d8)*(0.66666666666666663D-d9)*6D
				} else {
					r = d8
				}
				if 6D*d10 < 1.0D {
					g = d8 + (d7-d8)*6D*d10
				} else if 2D*d10 < 1.0D {
					g = d7
				} else if 3D*d10 < 2D {
					g = d8 + (d7-d8)*(0.66666666666666663D-d10)*6D
				} else {
					g = d8
				}
				if 6D*d11 < 1.0D {
					b = d8 + (d7-d8)*6D*d11
				} else if 2D*d11 < 1.0D {
					b = d7
				} else if 3D*d11 < 2D {
					b = d8 + (d7-d8)*(0.66666666666666663D-d11)*6D
				} else {
					b = d8
				}
			}
			byteR, ok := (r * 256D).(int)
			if !ok {
				panic("XXX Cast fail for *parser.GoCastType")
			}
			byteG, ok := (g * 256D).(int)
			if !ok {
				panic("XXX Cast fail for *parser.GoCastType")
			}
			byteB, ok := (b * 256D).(int)
			if !ok {
				panic("XXX Cast fail for *parser.GoCastType")
			}
			rgb := byteR<<16 + byteG<<8 + byteB
			rgb = Rasterizer3D.adjustBrightness(rgb, brightness)
			if rgb == 0 {
				rgb = 1
			}
			HslToRgb[++size] = rgb
		}
	}
	for textureId := 0; textureId < tEXTURE_LENGTH; textureId++ {
		if textures[textureId] != nil {
			originalPalette := <<unimp_obj.nm_*parser.GoArrayReference>>
			currentPalette[textureId] = make([]int, len(originalPalette))
			for colourId := 0; colourId < len(originalPalette); colourId++ {
				currentPalette[textureId][colourId] = adjustBrightness(originalPalette[colourId], brightness)
				if currentPalette[textureId][colourId]&0xf8f8ff == 0 && colourId != 0 {
					currentPalette[textureId][colourId] = 1
				}
			}
		}
	}
	for textureId := 0; textureId < tEXTURE_LENGTH; textureId++ {
		Rasterizer3D.RequestTextureUpdate(textureId)
	}
}
func SetDrawingArea(width int, length int) {
	scanOffsets = make([]int, length)
	for x := 0; x < length; x++ {
		scanOffsets[x] = width * x
	}
	OriginViewX = width / 2
	OriginViewY = length / 2
}

type ReferenceCache struct {
	empty      *Cacheable
	capacity   int
	table      *HashTable
	references *Queue
	spaceLeft  int
}

func NewReferenceCache(i int) (rcvr *ReferenceCache) {
	rcvr = &ReferenceCache{}
	rcvr.empty = NewCacheable()
	rcvr.references = NewQueue()
	rcvr.capacity = i
	rcvr.spaceLeft = i
	rcvr.table = NewHashTable()
	return
}
func (rcvr *ReferenceCache) Clear() {
	for {
		front := rcvr.references.popTail()
		if front != nil {
			front.unlink()
			front.unlinkCacheable()
		} else {
			rcvr.spaceLeft = rcvr.capacity
			return
		}
		if !(true) {
			break
		}
	}
}
func (rcvr *ReferenceCache) Get(key int64) (*Cacheable) {
	cacheable, ok := rcvr.table.get(key).(*Cacheable)
	if !ok {
		panic("XXX Cast fail for *parser.GoCastType")
	}
	if cacheable != nil {
		rcvr.references.insertHead(cacheable)
	}
	return cacheable
}
func (rcvr *ReferenceCache) Put(node *Cacheable, key int64) {
	if try() {
		if rcvr.spaceLeft == 0 {
			front := rcvr.references.popTail()
			front.unlink()
			front.unlinkCacheable()
			if front == rcvr.empty {
				front = rcvr.references.popTail()
				front.unlink()
				front.unlinkCacheable()
			}
		} else {
			rcvr.spaceLeft--
		}
		rcvr.table.put(node, key)
		rcvr.references.insertHead(node)
		return
	} else if catch_RuntimeException(runtimeexception) {
		fmt.Println(fmt.Sprintf("%v%v%v%v%v%v%v%v", "47547, ", node, ", ", key, ", ", 2 .(byte), ", ", runtimeexception.toString()))
	}
	throw(NewRuntimeException())
}

const fORCE_LOWEST_PLANE = 8
const bLOCKED_TILE = 1
const bRIDGE_TILE = 2

var maximumPlane int
var lowMem = false
var anInt131 int
var anIntArray140 []int
var anIntArray152 []int
var sINE_VERTICIES []int
var cOSINE_VERTICES []int

type Region struct {
	regionSizeX             int
	regionSizeY             int
	tileHeights             []int
	tileFlags               []byte
	hues                    []int
	saturations             []int
	luminances              []int
	chromas                 []int
	anIntArray128           []int
	shading                 []byte
	tileLighting            []int
	underlays               []byte
	overlays                []byte
	overlayTypes            []byte
	overlayOrientations     []byte
	anIntArrayArrayArray135 []int
}

func NewRegion(tileFlags []byte, tileHeights []int) (rcvr *Region) {
	rcvr = &Region{}
	rcvr.regionSizeX = 104
	rcvr.regionSizeY = 104
	rcvr.tileHeights = tileHeights
	rcvr.tileFlags = tileFlags
	rcvr.hues = make([]int, rcvr.regionSizeY)
	rcvr.saturations = make([]int, rcvr.regionSizeY)
	rcvr.luminances = make([]int, rcvr.regionSizeY)
	rcvr.chromas = make([]int, rcvr.regionSizeY)
	rcvr.anIntArray128 = make([]int, rcvr.regionSizeY)
	rcvr.shading = make([]byte, 4, rcvr.regionSizeX+1, rcvr.regionSizeY+1)
	rcvr.tileLighting = make([]int, rcvr.regionSizeX+1, rcvr.regionSizeY+1)
	rcvr.anIntArrayArrayArray135 = make([]int, 4, rcvr.regionSizeX+1, rcvr.regionSizeY+1)
	rcvr.underlays = make([]byte, 4, rcvr.regionSizeX, rcvr.regionSizeY)
	rcvr.overlays = make([]byte, 4, rcvr.regionSizeX, rcvr.regionSizeY)
	rcvr.overlayTypes = make([]byte, 4, rcvr.regionSizeX, rcvr.regionSizeY)
	rcvr.overlayOrientations = make([]byte, 4, rcvr.regionSizeX, rcvr.regionSizeY)
	maximumPlane = 99
	return
}
func calculateNoise(x int, y int) (int) {
	k := x + y*57
	k = k<<13 ^ k
	l := (k*(k*k*15731+0xc0ae5) + 0x5208dd0d) & 0x7fffffff
	return uint32(l) >> 19 & 0xff
}
func calculateVertexHeight(i int, j int) (int) {
	mapHeight := Region.interpolatedNoise(i+45365, j+0x16713, 4) - 128 + uint32(Region.interpolatedNoise(i+10294, j+37821, 2)-128)>>1 + uint32(Region.interpolatedNoise(i, j, 1)-128)>>2
	mapHeight = (mapHeight.(float64) * 0.29999999999999999D).(int) + 35
	if mapHeight < 10 {
		mapHeight = 10
	} else if mapHeight > 60 {
		mapHeight = 60
	}
	return mapHeight
}
func (rcvr *Region) checkedLight(color int, light int) (int) {
	if color == -2 {
		return 0xbc614e
	}
	if color == -1 {
		if light < 0 {
			light = 0
		} else if light > 127 {
			light = 127
		}
		light = 127 - light
		return light
	}
	light = light * (color & 0x7f) / 128
	if light < 2 {
		light = 2
	} else if light > 126 {
		light = 126
	}
	return color&0xff80 + light
}
func (rcvr *Region) CreateRegionScene(maps []*CollisionMap, scene *SceneGraph) {
	if try() {
		for z := 0; z < 4; z++ {
			for x := 0; x < 104; x++ {
				for y := 0; y < 104; y++ {
					if tileFlags[z][x][y]&bLOCKED_TILE == 1 {
						plane := z
						if tileFlags[1][x][y]&bRIDGE_TILE == 2 {
							plane--
						}
						if plane >= 0 {
							maps[plane].block(x, y)
						}
					}
				}
			}
		}
		for z := 0; z < 4; z++ {
			shading := rcvr.shading[z]
			byte0 := 96
			diffusion := 'a'
			lightX := -50
			lightY := -10
			lightZ := -50
			light, ok := Math.sqrt(lightX*lightX + lightY*lightY + lightZ*lightZ).(int)
			if !ok {
				panic("XXX Cast fail for *parser.GoCastType")
			}
			l3 := uint32(diffusion*light) >> 8
			for j4 := 1; j4 < rcvr.regionSizeY-1; j4++ {
				for j5 := 1; j5 < rcvr.regionSizeX-1; j5++ {
					k6 := tileHeights[z][j5+1][j4] - tileHeights[z][j5-1][j4]
					l7 := tileHeights[z][j5][j4+1] - tileHeights[z][j5][j4-1]
					j9, ok := Math.sqrt(k6*k6 + 0x10000 + l7*l7).(int)
					if !ok {
						panic("XXX Cast fail for *parser.GoCastType")
					}
					k12 := k6 << 8 / j9
					l13 := 0x10000 / j9
					j15 := l7 << 8 / j9
					j16 := byte0 + (lightX*k12+lightY*l13+lightZ*j15)/l3
					j17 := uint32(shading[j5-1][j4])>>2 + uint32(shading[j5+1][j4])>>3 + uint32(shading[j5][j4-1])>>2 + uint32(shading[j5][j4+1])>>3 + uint32(shading[j5][j4])>>1
					tileLighting[j5][j4] = j16 - j17
				}
			}
			for k5 := 0; k5 < rcvr.regionSizeY; k5++ {
				hues[k5] = 0
				saturations[k5] = 0
				luminances[k5] = 0
				chromas[k5] = 0
				anIntArray128[k5] = 0
			}
			for l6 := -5; l6 < rcvr.regionSizeX+5; l6++ {
				for i8 := 0; i8 < rcvr.regionSizeY; i8++ {
					k9 := l6 + 5
					if k9 >= 0 && k9 < rcvr.regionSizeX {
						l12 := underlays[z][k9][i8] & 0xff
						if l12 > 0 {
							if l12 > FloorDefinition.underlays.length {
								l12 = FloorDefinition.underlays.length
							}
							flo := <<unimp_arrayref_FloorDefinition.underlays>>[l12-1]
							hues[i8] += flo.blendHue
							saturations[i8] += flo.saturation
							luminances[i8] += flo.luminance
							chromas[i8] += flo.blendHueMultiplier
							anIntArray128[i8]++
						}
					}
					i13 := l6 - 5
					if i13 >= 0 && i13 < rcvr.regionSizeX {
						i14 := underlays[z][i13][i8] & 0xff
						if i14 > 0 {
							flo_1 := <<unimp_arrayref_FloorDefinition.underlays>>[i14-1]
							hues[i8] -= flo_1.blendHue
							saturations[i8] -= flo_1.saturation
							luminances[i8] -= flo_1.luminance
							chromas[i8] -= flo_1.blendHueMultiplier
							anIntArray128[i8]--
						}
					}
				}
				if l6 >= 1 && l6 < rcvr.regionSizeX-1 {
					l9 := 0
					j13 := 0
					j14 := 0
					k15 := 0
					k16 := 0
					for k17 := -5; k17 < rcvr.regionSizeY+5; k17++ {
						j18 := k17 + 5
						if j18 >= 0 && j18 < rcvr.regionSizeY {
							l9 += hues[j18]
							j13 += saturations[j18]
							j14 += luminances[j18]
							k15 += chromas[j18]
							k16 += anIntArray128[j18]
						}
						k18 := k17 - 5
						if k18 >= 0 && k18 < rcvr.regionSizeY {
							l9 -= hues[k18]
							j13 -= saturations[k18]
							j14 -= luminances[k18]
							k15 -= chromas[k18]
							k16 -= anIntArray128[k18]
						}
						if k17 >= 1 && k17 < rcvr.regionSizeY-1 && (!lowMem || tileFlags[0][l6][k17]&2 != 0 || tileFlags[z][l6][k17]&0x10 == 0 && rcvr.getCollisionPlane(k17, z, l6) == anInt131) {
							if z < maximumPlane {
								maximumPlane = z
							}
							l18 := underlays[z][l6][k17] & 0xff
							i19 := overlays[z][l6][k17] & 0xff
							if l18 > 0 || i19 > 0 {
								j19 := tileHeights[z][l6][k17]
								k19 := tileHeights[z][l6+1][k17]
								l19 := tileHeights[z][l6+1][k17+1]
								i20 := tileHeights[z][l6][k17+1]
								j20 := tileLighting[l6][k17]
								k20 := tileLighting[l6+1][k17]
								l20 := tileLighting[l6+1][k17+1]
								i21 := tileLighting[l6][k17+1]
								j21 := -1
								k21 := -1
								if l18 > 0 {
									l21 := l9 * 256 / k15
									j22 := j13 / k16
									l22 := j14 / k16
									j21 = rcvr.encode(l21, j22, l22)
									if l22 < 0 {
										l22 = 0
									} else if l22 > 255 {
										l22 = 255
									}
									k21 = rcvr.encode(l21, j22, l22)
								}
								if z > 0 {
									flag := true
									if l18 == 0 && overlayTypes[z][l6][k17] != 0 {
										flag = false
									}
									if i19 > 0 && !<<unimp_obj.nm_*parser.GoArrayReference>> {
										flag = false
									}
									if flag && j19 == k19 && j19 == l19 && j19 == i20 {
										anIntArrayArrayArray135[z][l6][k17] |= 0x924
									}
								}
								i22 := 0
								if j21 != -1 {
									i22 = <<unimp_arrayref_Rasterizer3D.hslToRgb>>[Region.method187(k21, 96)]
								}
								if i19 == 0 {
									scene.addTile(z, l6, k17, 0, 0, -1, j19, k19, l19, i20, Region.method187(j21, j20), Region.method187(j21, k20), Region.method187(j21, l20), Region.method187(j21, i21), 0, 0, 0, 0, i22, 0)
								} else {
									k22 := overlayTypes[z][l6][k17] + 1
									byte4 := overlayOrientations[z][l6][k17]
									if i19-1 > FloorDefinition.overlays.length {
										i19 = FloorDefinition.overlays.length
									}
									overlay_flo := <<unimp_arrayref_FloorDefinition.overlays>>[i19-1]
									textureId := overlay_flo.texture
									var j23 int
									var minimapColor int
									if textureId >= 0 {
										minimapColor = Rasterizer3D.GetOverallColour(textureId)
										j23 = -1
									} else if overlay_flo.rgb == 0xff00ff {
										minimapColor = 0
										j23 = -2
										textureId = -1
									} else if overlay_flo.rgb == 0x333333 {
										minimapColor = <<unimp_arrayref_Rasterizer3D.hslToRgb>>[rcvr.checkedLight(overlay_flo.hsl16, 96)]
										j23 = -2
										textureId = -1
									} else {
										j23 = encode(overlay_flo.hue, overlay_flo.saturation, overlay_flo.luminance)
										minimapColor = <<unimp_arrayref_Rasterizer3D.hslToRgb>>[rcvr.checkedLight(overlay_flo.hsl16, 96)]
									}
									if minimapColor == 0x000000 && overlay_flo.anotherRgb != -1 {
										newMinimapColor := encode(overlay_flo.anotherHue, overlay_flo.anotherSaturation, overlay_flo.anotherLuminance)
										minimapColor = <<unimp_arrayref_Rasterizer3D.hslToRgb>>[checkedLight(newMinimapColor, 96)]
									}
									scene.addTile(z, l6, k17, k22, byte4, textureId, j19, k19, l19, i20, Region.method187(j21, j20), Region.method187(j21, k20), Region.method187(j21, l20), Region.method187(j21, i21), checkedLight(j23, j20), checkedLight(j23, k20), checkedLight(j23, l20), checkedLight(j23, i21), i22, minimapColor)
								}
							}
						}
					}
				}
			}
			for j8 := 1; j8 < rcvr.regionSizeY-1; j8++ {
				for i10 := 1; i10 < rcvr.regionSizeX-1; i10++ {
					scene.setTileLogicHeight(z, i10, j8, rcvr.getCollisionPlane(j8, z, i10))
				}
			}
		}
		scene.shadeModels(-10, -50, -50)
		for j1 := 0; j1 < rcvr.regionSizeX; j1++ {
			for l1 := 0; l1 < rcvr.regionSizeY; l1++ {
				if tileFlags[1][j1][l1]&2 == 2 {
					scene.applyBridgeMode(l1, j1)
				}
			}
		}
		i2 := 1
		j2 := 2
		k2 := 4
		for l2 := 0; l2 < 4; l2++ {
			if l2 > 0 {
				i2 = i2 << 3
				j2 = j2 << 3
				k2 = k2 << 3
			}
			for i3 := 0; i3 <= l2; i3++ {
				for k3 := 0; k3 <= rcvr.regionSizeY; k3++ {
					for i4 := 0; i4 <= rcvr.regionSizeX; i4++ {
						if anIntArrayArrayArray135[i3][i4][k3]&i2 != 0 {
							k4 := k3
							l5 := k3
							i7 := i3
							k8 := i3
							for ; k4 > 0 && anIntArrayArrayArray135[i3][i4][k4-1]&i2 != 0; --k4 {
								<<unimp_stmt[*grammar.JEmpty]>>
							}
							for ; l5 < rcvr.regionSizeY && anIntArrayArrayArray135[i3][i4][l5+1]&i2 != 0; ++l5 {
								<<unimp_stmt[*grammar.JEmpty]>>
							}
						label0:
							<<unimp_stmt[*grammar.JForExpr]>>
						label1:
							<<unimp_stmt[*grammar.JForExpr]>>
							l10 := (k8 + 1 - i7) * (l5 - k4 + 1)
							if l10 >= 8 {
								c1 := '\360'
								k14 := tileHeights[k8][i4][k4] - c1
								l15 := tileHeights[i7][i4][k4]
								SceneGraph.CreateNewSceneCluster(l2, i4*128, l15, i4*128, l5*128+128, k14, k4*128, 1)
								for l16 := i7; l16 <= k8; l16++ {
									for l17 := k4; l17 <= l5; l17++ {
										anIntArrayArrayArray135[l16][i4][l17] &= ^i2
									}
								}
							}
						}
						if anIntArrayArrayArray135[i3][i4][k3]&j2 != 0 {
							l4 := i4
							i6 := i4
							j7 := i3
							l8 := i3
							for ; l4 > 0 && anIntArrayArrayArray135[i3][l4-1][k3]&j2 != 0; --l4 {
								<<unimp_stmt[*grammar.JEmpty]>>
							}
							for ; i6 < rcvr.regionSizeX && anIntArrayArrayArray135[i3][i6+1][k3]&j2 != 0; ++i6 {
								<<unimp_stmt[*grammar.JEmpty]>>
							}
						label2:
							<<unimp_stmt[*grammar.JForExpr]>>
						label3:
							<<unimp_stmt[*grammar.JForExpr]>>
							k11 := (l8 + 1 - j7) * (i6 - l4 + 1)
							if k11 >= 8 {
								c2 := '\360'
								l14 := tileHeights[l8][l4][k3] - c2
								i16 := tileHeights[j7][l4][k3]
								SceneGraph.CreateNewSceneCluster(l2, l4*128, i16, i6*128+128, k3*128, l14, k3*128, 2)
								for i17 := j7; i17 <= l8; i17++ {
									for i18 := l4; i18 <= i6; i18++ {
										anIntArrayArrayArray135[i17][i18][k3] &= ^j2
									}
								}
							}
						}
						if anIntArrayArrayArray135[i3][i4][k3]&k2 != 0 {
							i5 := i4
							j6 := i4
							k7 := k3
							i9 := k3
							for ; k7 > 0 && anIntArrayArrayArray135[i3][i4][k7-1]&k2 != 0; --k7 {
								<<unimp_stmt[*grammar.JEmpty]>>
							}
							for ; i9 < rcvr.regionSizeY && anIntArrayArrayArray135[i3][i4][i9+1]&k2 != 0; ++i9 {
								<<unimp_stmt[*grammar.JEmpty]>>
							}
						label4:
							<<unimp_stmt[*grammar.JForExpr]>>
						label5:
							<<unimp_stmt[*grammar.JForExpr]>>
							if (j6-i5+1)*(i9-k7+1) >= 4 {
								j12 := tileHeights[i3][i5][k7]
								SceneGraph.CreateNewSceneCluster(l2, i5*128, j12, j6*128+128, i9*128+128, j12, k7*128, 4)
								for k13 := i5; k13 <= j6; k13++ {
									for i15 := k7; i15 <= i9; i15++ {
										anIntArrayArrayArray135[i3][k13][i15] &= ^k2
									}
								}
							}
						}
					}
				}
			}
		}
	} else if catch_Exception(e) {
		e.printStackTrace()
	}
}
func (rcvr *Region) encode(hue int, saturation int, luminance int) (int) {
	if luminance > 179 {
		saturation /= 2
	}
	if luminance > 192 {
		saturation /= 2
	}
	if luminance > 217 {
		saturation /= 2
	}
	if luminance > 243 {
		saturation /= 2
	}
	return hue/4<<10 + saturation/32<<7 + luminance/2
}
func (rcvr *Region) getCollisionPlane(y int, z int, x int) (int) {
	if tileFlags[z][x][y]&fORCE_LOWEST_PLANE != 0 {
		return 0
	}
	if z > 0 && tileFlags[1][x][y]&bRIDGE_TILE != 0 {
		return z - 1
	} else {
		return z
	}
}
func (rcvr *Region) InitiateVertexHeights(yOffset int, yLength int, xLength int, xOffset int) {
	for y := yOffset; y <= yOffset+yLength; y++ {
		for x := xOffset; x <= xOffset+xLength; x++ {
			if x >= 0 && x < rcvr.regionSizeX && y >= 0 && y < rcvr.regionSizeY {
				shading[0][x][y] = 127
				if x == xOffset && x > 0 {
					tileHeights[0][x][y] = tileHeights[0][x-1][y]
				}
				if x == xOffset+xLength && x < rcvr.regionSizeX-1 {
					tileHeights[0][x][y] = tileHeights[0][x+1][y]
				}
				if y == yOffset && y > 0 {
					tileHeights[0][x][y] = tileHeights[0][x][y-1]
				}
				if y == yOffset+yLength && y < rcvr.regionSizeY-1 {
					tileHeights[0][x][y] = tileHeights[0][x][y+1]
				}
			}
		}
	}
}
func interpolate(a int, b int, angle int, frequencyReciprocal int) (int) {
	cosine := uint32(0x10000-<<unimp_arrayref_Rasterizer3D.cosine>>[angle*1024/frequencyReciprocal]) >> 1
	return uint32(a*(0x10000-cosine))>>16 + uint32(b*cosine)>>16
}
func interpolatedNoise(x int, y int, frequencyReciprocal int) (int) {
	l := x / frequencyReciprocal
	i1 := x & (frequencyReciprocal - 1)
	j1 := y / frequencyReciprocal
	k1 := y & (frequencyReciprocal - 1)
	l1 := Region.smoothNoise(l, j1)
	i2 := Region.smoothNoise(l+1, j1)
	j2 := Region.smoothNoise(l, j1+1)
	k2 := Region.smoothNoise(l+1, j1+1)
	l2 := Region.interpolate(l1, i2, i1, frequencyReciprocal)
	i3 := Region.interpolate(j2, k2, i1, frequencyReciprocal)
	return Region.interpolate(l2, i3, k1, frequencyReciprocal)
}
func (rcvr *Region) Method180(abyte0 []byte, i int, j int, k int, l int, aclass11 []*CollisionMap) {
	for i1 := 0; i1 < 4; i1++ {
		for j1 := 0; j1 < 64; j1++ {
			for k1 := 0; k1 < 64; k1++ {
				if j+j1 > 0 && j+j1 < 103 && i+k1 > 0 && i+k1 < 103 {
					<<unimp_obj.nm_*parser.GoArrayReference>>[j+j1][i+k1] &= 0xfeffffff
				}
			}
		}
	}
	stream := NewBuffer(abyte0)
	for l1 := 0; l1 < 4; l1++ {
		for i2 := 0; i2 < 64; i2++ {
			for j2 := 0; j2 < 64; j2++ {
				rcvr.readTile(j2+i, l, stream, i2+j, l1, 0, k)
			}
		}
	}
}
func method187(i int, j int) (int) {
	if i == -1 {
		return 0xbc614e
	}
	j = j * (i & 0x7f) / 128
	if j < 2 {
		j = 2
	} else if j > 126 {
		j = 126
	}
	return i&0xff80 + j
}
func (rcvr *Region) Method190(i int, aclass11 []*CollisionMap, j int, worldController *SceneGraph, abyte0 []byte) {
label0:
	{
		stream := NewBuffer(abyte0)
		l := -1
		for {
			i1 := stream.getUIncrementalSmart()
			if i1 == 0 {
				break label0
			}
			l += i1
			j1 := 0
			for {
				k1 := stream.readUSmart()
				if k1 == 0 {
					break
				}
				j1 += k1 - 1
				l1 := j1 & 0x3f
				i2 := uint32(j1) >> 6 & 0x3f
				j2 := uint32(j1) >> 12
				k2 := stream.readUnsignedByte()
				l2 := uint32(k2) >> 2
				i3 := k2 & 3
				j3 := i2 + i
				k3 := l1 + j
				if j3 > 0 && k3 > 0 && j3 < 103 && k3 < 103 && j2 >= 0 && j2 < 4 {
					l3 := j2
					if tileFlags[1][j3][k3]&2 == 2 {
						l3--
					}
					class11 := nil
					if l3 >= 0 {
						class11 = aclass11[l3]
					}
					rcvr.renderObject(k3, worldController, class11, l2, j2, j3, l, i3)
				}
				if !(true) {
					break
				}
			}
			if !(true) {
				break
			}
		}
	}
}
func (rcvr *Region) readTile(i int, j int, stream *Buffer, k int, l int, i1 int, k1 int) {
	if try() {
		if k >= 0 && k < 104 && i >= 0 && i < 104 {
			tileFlags[l][k][i] = 0
			for {
				l1 := stream.readUnsignedByte()
				if l1 == 0 {
					if l == 0 {
						tileHeights[0][k][i] = -Region.calculateVertexHeight(0xe3b7b+k+k1, 0x87cce+i+j) * 8
						return
					} else {
						tileHeights[l][k][i] = tileHeights[l-1][k][i] - 240
						return
					}
				}
				if l1 == 1 {
					j2 := stream.readUnsignedByte()
					if j2 == 1 {
						j2 = 0
					}
					if l == 0 {
						tileHeights[0][k][i] = -j2 * 8
						return
					} else {
						tileHeights[l][k][i] = tileHeights[l-1][k][i] - j2*8
						return
					}
				}
				if l1 <= 49 {
					overlays[l][k][i] = stream.readSignedByte()
					overlayTypes[l][k][i] = ((l1 - 2) / 4).(byte)
					overlayOrientations[l][k][i] = ((l1 - 2 + i1) & 3).(byte)
				} else if l1 <= 81 {
					tileFlags[l][k][i] = (l1 - 49).(byte)
				} else {
					underlays[l][k][i] = (l1 - 81).(byte)
				}
				if !(true) {
					break
				}
			}
		}
		for {
			i2 := stream.readUnsignedByte()
			if i2 == 0 {
				break
			}
			if i2 == 1 {
				stream.readUnsignedByte()
				return
			}
			if i2 <= 49 {
				stream.readUnsignedByte()
			}
			if !(true) {
				break
			}
		}
	} else if catch_Exception(e) {
	}
}
func (rcvr *Region) renderObject(y int, scene *SceneGraph, class11 *CollisionMap, type int, z int, x int, id int, j1 int) {
	if lowMem && tileFlags[0][x][y]&bRIDGE_TILE == 0 {
		if tileFlags[z][x][y]&0x10 != 0 {
			return
		}
		if rcvr.getCollisionPlane(y, z, x) != anInt131 {
			return
		}
	}
	if z < maximumPlane {
		maximumPlane = z
	}
	definition := ObjectDefinition.Get(id)
	var sizeY int
	var sizeX int
	if j1 != 1 && j1 != 3 {
		sizeX = definition.objectSizeX
		sizeY = definition.objectSizeY
	} else {
		sizeX = definition.objectSizeY
		sizeY = definition.objectSizeX
	}
	var editX int
	var editX2 int
	if x+sizeX <= 104 {
		editX = x + uint32(sizeX)>>1
		editX2 = x + uint32(1+sizeX)>>1
	} else {
		editX = x
		editX2 = 1 + x
	}
	var editY int
	var editY2 int
	if sizeY+y <= 104 {
		editY = uint32(sizeY)>>1 + y
		editY2 = y + uint32(1+sizeY)>>1
	} else {
		editY = y
		editY2 = 1 + y
	}
	center := tileHeights[z][editX][editY]
	east := tileHeights[z][editX2][editY]
	northEast := tileHeights[z][editX2][editY2]
	north := tileHeights[z][editX][editY2]
	mean := uint32(center+east+northEast+north) >> 2
	key := x + y<<7 + id<<14 + 0x40000000
	if !definition.isInteractive {
		key += 0x80000000
	}
	config, ok := (j1<<6 + type).(byte)
	if !ok {
		panic("XXX Cast fail for *parser.GoCastType")
	}
	if type == 22 {
		if lowMem && !definition.isInteractive && !definition.obstructsGround {
			return
		}
		var obj *Object
		if definition.childrenIds == nil {
			obj = definition.modelAt(22, j1, center, east, northEast, north)
		} else {
			obj = NewSceneObject(id, j1, 22, east, northEast, center, north)
		}
		scene.addGroundDecoration(z, mean, y, obj.(*Renderable), config, key, x)
		if definition.solid && definition.isInteractive && class11 != nil {
			class11.block(x, y)
		}
		return
	}
	if type == 10 || type == 11 {
		var obj1 *Object
		if definition.childrenIds == nil {
			obj1 = definition.modelAt(10, j1, center, east, northEast, north)
		} else {
			obj1 = NewSceneObject(id, j1, 10, east, northEast, center, north)
		}
		if obj1 != nil {
			i5 := 0
			if type == 11 {
				i5 += 256
			}
			var j4 int
			var l4 int
			if j1 == 1 || j1 == 3 {
				j4 = definition.objectSizeY
				l4 = definition.objectSizeX
			} else {
				j4 = definition.objectSizeX
				l4 = definition.objectSizeY
			}
			if scene.addTiledObject(key, config, mean, l4, obj1.(*Renderable), j4, z, i5, y, x) && definition.castsShadow {
				var model *Model
				if _, ok := obj1.(Model); ok {
					model = obj1.(*Model)
				} else {
					model = definition.modelAt(10, j1, center, east, northEast, north)
				}
				if model != nil {
					for j5 := 0; j5 <= j4; j5++ {
						for k5 := 0; k5 <= l4; k5++ {
							l5 := model.maxVertexDistanceXZPlane / 4
							if l5 > 30 {
								l5 = 30
							}
							if l5 > shading[z][x+j5][y+k5] {
								shading[z][x+j5][y+k5] = l5.(byte)
							}
						}
					}
				}
			}
		}
		if definition.solid && class11 != nil {
			class11.markInteractiveObject(x, y, j1, definition.objectSizeX, definition.objectSizeY, definition.walkable)
		}
		return
	}
	if type >= 12 {
		var obj2 *Object
		if definition.childrenIds == nil {
			obj2 = definition.modelAt(type, j1, center, east, northEast, north)
		} else {
			obj2 = NewSceneObject(id, j1, type, east, northEast, center, north)
		}
		scene.addTiledObject(key, config, mean, 1, obj2.(*Renderable), 1, z, 0, y, x)
		if type >= 12 && type <= 17 && type != 13 && z > 0 {
			anIntArrayArrayArray135[z][x][y] |= 0x924
		}
		if definition.solid && class11 != nil {
			class11.markInteractiveObject(x, y, j1, definition.objectSizeX, definition.objectSizeY, definition.walkable)
		}
		return
	}
	if type == 0 {
		var obj3 *Object
		if definition.childrenIds == nil {
			obj3 = definition.modelAt(0, j1, center, east, northEast, north)
		} else {
			obj3 = NewSceneObject(id, j1, 0, east, northEast, center, north)
		}
		scene.addWallObject(anIntArray152[j1], obj3.(*Renderable), key, y, config, x, nil, mean, 0, z)
		if j1 == 0 {
			if definition.castsShadow {
				shading[z][x][y] = 50
				shading[z][x][y+1] = 50
			}
			if definition.occludes {
				anIntArrayArrayArray135[z][x][y] |= 0x249
			}
		} else if j1 == 1 {
			if definition.castsShadow {
				shading[z][x][y+1] = 50
				shading[z][x+1][y+1] = 50
			}
			if definition.occludes {
				anIntArrayArrayArray135[z][x][y+1] |= 0x492
			}
		} else if j1 == 2 {
			if definition.castsShadow {
				shading[z][x+1][y] = 50
				shading[z][x+1][y+1] = 50
			}
			if definition.occludes {
				anIntArrayArrayArray135[z][x+1][y] |= 0x249
			}
		} else if j1 == 3 {
			if definition.castsShadow {
				shading[z][x][y] = 50
				shading[z][x+1][y] = 50
			}
			if definition.occludes {
				anIntArrayArrayArray135[z][x][y] |= 0x492
			}
		}
		if definition.solid && class11 != nil {
			class11.markWall(x, y, type, j1, definition.walkable)
		}
		if definition.decorDisplacement != 16 {
			scene.method290(y, definition.decorDisplacement, x, z)
		}
		return
	}
	if type == 1 {
		var obj4 *Object
		if definition.childrenIds == nil {
			obj4 = definition.modelAt(1, j1, center, east, northEast, north)
		} else {
			obj4 = NewSceneObject(id, j1, 1, east, northEast, center, north)
		}
		scene.addWallObject(anIntArray140[j1], obj4.(*Renderable), key, y, config, x, nil, mean, 0, z)
		if definition.castsShadow {
			if j1 == 0 {
				shading[z][x][y+1] = 50
			} else if j1 == 1 {
				shading[z][x+1][y+1] = 50
			} else if j1 == 2 {
				shading[z][x+1][y] = 50
			} else if j1 == 3 {
				shading[z][x][y] = 50
			}
		}
		if definition.solid && class11 != nil {
			class11.markWall(x, y, type, j1, definition.walkable)
		}
		return
	}
	if type == 2 {
		i3 := (j1 + 1) & 3
		var obj11 *Object
		var obj12 *Object
		if definition.childrenIds == nil {
			obj11 = definition.modelAt(2, 4+j1, center, east, northEast, north)
			obj12 = definition.modelAt(2, i3, center, east, northEast, north)
		} else {
			obj11 = NewSceneObject(id, 4+j1, 2, east, northEast, center, north)
			obj12 = NewSceneObject(id, i3, 2, east, northEast, center, north)
		}
		scene.addWallObject(anIntArray152[j1], obj11.(*Renderable), key, y, config, x, obj12.(*Renderable), mean, anIntArray152[i3], z)
		if definition.occludes {
			if j1 == 0 {
				anIntArrayArrayArray135[z][x][y] |= 0x249
				anIntArrayArrayArray135[z][x][y+1] |= 0x492
			} else if j1 == 1 {
				anIntArrayArrayArray135[z][x][y+1] |= 0x492
				anIntArrayArrayArray135[z][x+1][y] |= 0x249
			} else if j1 == 2 {
				anIntArrayArrayArray135[z][x+1][y] |= 0x249
				anIntArrayArrayArray135[z][x][y] |= 0x492
			} else if j1 == 3 {
				anIntArrayArrayArray135[z][x][y] |= 0x492
				anIntArrayArrayArray135[z][x][y] |= 0x249
			}
		}
		if definition.solid && class11 != nil {
			class11.markWall(x, y, type, j1, definition.walkable)
		}
		if definition.decorDisplacement != 16 {
			scene.method290(y, definition.decorDisplacement, x, z)
		}
		return
	}
	if type == 3 {
		var obj5 *Object
		if definition.childrenIds == nil {
			obj5 = definition.modelAt(3, j1, center, east, northEast, north)
		} else {
			obj5 = NewSceneObject(id, j1, 3, east, northEast, center, north)
		}
		scene.addWallObject(anIntArray140[j1], obj5.(*Renderable), key, y, config, x, nil, mean, 0, z)
		if definition.castsShadow {
			if j1 == 0 {
				shading[z][x][y+1] = 50
			} else if j1 == 1 {
				shading[z][x+1][y+1] = 50
			} else if j1 == 2 {
				shading[z][x+1][y] = 50
			} else if j1 == 3 {
				shading[z][x][y] = 50
			}
		}
		if definition.solid && class11 != nil {
			class11.markWall(x, y, type, j1, definition.walkable)
		}
		return
	}
	if type == 9 {
		var obj6 *Object
		if definition.childrenIds == nil {
			obj6 = definition.modelAt(type, j1, center, east, northEast, north)
		} else {
			obj6 = NewSceneObject(id, j1, type, east, northEast, center, north)
		}
		scene.addTiledObject(key, config, mean, 1, obj6.(*Renderable), 1, z, 0, y, x)
		if definition.solid && class11 != nil {
			class11.markInteractiveObject(x, y, j1, definition.objectSizeX, definition.objectSizeY, definition.walkable)
		}
		return
	}
	if definition.contouredGround {
		if j1 == 1 {
			j3 := north
			north = northEast
			northEast = east
			east = center
			center = j3
		} else if j1 == 2 {
			k3 := north
			north = east
			east = k3
			k3 = northEast
			northEast = center
			center = k3
		} else if j1 == 3 {
			l3 := north
			north = center
			center = east
			east = northEast
			northEast = l3
		}
	}
	if type == 4 {
		var obj7 *Object
		if definition.childrenIds == nil {
			obj7 = definition.modelAt(4, 0, center, east, northEast, north)
		} else {
			obj7 = NewSceneObject(id, 0, 4, east, northEast, center, north)
		}
		scene.addWallDecoration(key, y, j1*512, z, 0, mean, obj7.(*Renderable), x, config, 0, anIntArray152[j1])
		return
	}
	if type == 5 {
		i4 := 16
		k4 := scene.getWallObjectUid(z, x, y)
		if k4 > 0 {
			i4 = <<unimp_obj.nm_*parser.GoMethodAccess>>
		}
		var obj13 *Object
		if definition.childrenIds == nil {
			obj13 = definition.modelAt(4, 0, center, east, northEast, north)
		} else {
			obj13 = NewSceneObject(id, 0, 4, east, northEast, center, north)
		}
		scene.addWallDecoration(key, y, j1*512, z, cOSINE_VERTICES[j1]*i4, mean, obj13.(*Renderable), x, config, sINE_VERTICIES[j1]*i4, anIntArray152[j1])
		return
	}
	if type == 6 {
		var obj8 *Object
		if definition.childrenIds == nil {
			obj8 = definition.modelAt(4, 0, center, east, northEast, north)
		} else {
			obj8 = NewSceneObject(id, 0, 4, east, northEast, center, north)
		}
		scene.addWallDecoration(key, y, j1, z, 0, mean, obj8.(*Renderable), x, config, 0, 256)
		return
	}
	if type == 7 {
		var obj9 *Object
		if definition.childrenIds == nil {
			obj9 = definition.modelAt(4, 0, center, east, northEast, north)
		} else {
			obj9 = NewSceneObject(id, 0, 4, east, northEast, center, north)
		}
		scene.addWallDecoration(key, y, j1, z, 0, mean, obj9.(*Renderable), x, config, 0, 512)
		return
	}
	if type == 8 {
		var obj10 *Object
		if definition.childrenIds == nil {
			obj10 = definition.modelAt(4, 0, center, east, northEast, north)
		} else {
			obj10 = NewSceneObject(id, 0, 4, east, northEast, center, north)
		}
		scene.addWallDecoration(key, y, j1, z, 0, mean, obj10.(*Renderable), x, config, 0, 768)
	}
}
func smoothNoise(x int, y int) (int) {
	corners := Region.calculateNoise(x-1, y-1) + Region.calculateNoise(x+1, y-1) + Region.calculateNoise(x-1, y+1) + Region.calculateNoise(x+1, y+1)
	sides := Region.calculateNoise(x-1, y) + Region.calculateNoise(x+1, y) + Region.calculateNoise(x, y-1) + Region.calculateNoise(x, y+1)
	center := Region.calculateNoise(x, y)
	return corners/16 + sides/8 + center/4
}

type Renderable struct {
	*Cacheable
	ModelBaseY    int
	VertexNormals []*VertexNormal
}

func NewRenderable() (rcvr *Renderable) {
	rcvr = &Renderable{}
	return
}
func (rcvr *Renderable) GetRotatedModel() (*Model) {
	return nil
}
func (rcvr *Renderable) RenderAtPoint(i int, j int, k int, l int, i1 int, j1 int, k1 int, l1 int, i2 int) {
	model := rcvr.GetRotatedModel()
	if model != nil {
		rcvr.ModelBaseY = model.modelBaseY
		model.renderAtPoint(i, j, k, l, i1, j1, k1, l1, i2)
	}
}

var interactableObjects = make([]*GameObject, 100)
var cullingClusterPlaneCount = 4
var sceneClusters = make([]*SceneCluster, cullingClusterPlaneCount, 500)
var viewportHalfWidth int
var viewportHalfHeight int
var anInt495 int
var anInt496 int
var viewportWidth int
var viewportHeight int
var camUpDownY int
var camUpDownX int
var camLeftRightY int
var camLeftRightX int
var aBooleanArrayArrayArrayArray491 = make([]bool, 8, 32, 51, 51)
var sceneClusterCounts = make([]int, cullingClusterPlaneCount)
var ViewDistance = 9
var renderedObjectCount int
var xCameraTile int
var yCameraTile int
var xCameraPos int
var zCameraPos int
var yCameraPos int
var currentRenderPlane int
var cameraLowTileX int
var cameraHighTileX int
var cameraLowTileY int
var cameraHighTileY int
var clickCount int
var clicked bool
var tileDeque = NewDeque()
var anIntArray478 []int
var anIntArray479 []int
var anIntArray480 []int
var anIntArray481 []int
var anIntArray482 []int
var anIntArray483 []int
var anIntArray484 []int
var anIntArray463 []int
var anIntArray464 []int
var anIntArray465 []int
var anIntArray466 []int
var clickedTileX = -1
var clickedTileY = -1
var clickScreenX int
var clickScreenY int
var tEXTURE_COLORS []int
var fixedCullingClusters = make([]*SceneCluster, 500)
var processedCullingCluster int

type SceneGraph struct {
	gameObjectCache                []*GameObject
	tileArray                      []*Tile
	heightMap                      []int
	zRegionSize                    int
	xRegionSize                    int
	yRegionSize                    int
	interactableObjectCacheCurrPos int
	cameraLowTileZ                 int
	renderedViewableObjects        []int
	mergeANormals                  []int
	mergeBNormals                  []int
	mergeNormalsIndex              int
}

func NewSceneGraph(heightMap []int) (rcvr *SceneGraph) {
	rcvr = &SceneGraph{}
	xLocSize := 104
	yLocSize := 104
	zLocSize := 4
	rcvr.gameObjectCache = make([]*GameObject, 5000)
	rcvr.mergeANormals = make([]int, 10000)
	rcvr.mergeBNormals = make([]int, 10000)
	rcvr.zRegionSize = zLocSize
	rcvr.xRegionSize = xLocSize
	rcvr.yRegionSize = yLocSize
	rcvr.tileArray = make([]*Tile, zLocSize, xLocSize, yLocSize)
	rcvr.renderedViewableObjects = make([]int, zLocSize, xLocSize+1, yLocSize+1)
	rcvr.heightMap = heightMap
	rcvr.InitializeToNull()
	return
}
func (rcvr *SceneGraph) addAnimableC(zLoc int, xLoc int, yLoc int, sizeX int, sizeY int, xPos int, yPos int, tileHeight int, renderable *Renderable, turnValue int, isDynamic bool, uid int, objectRotationType byte) (bool) {
	for x := xLoc; x < xLoc+sizeX; x++ {
		for y := yLoc; y < yLoc+sizeY; y++ {
			if x < 0 || y < 0 || x >= rcvr.xRegionSize || y >= rcvr.yRegionSize {
				return false
			}
			tile := tileArray[zLoc][x][y]
			if tile != nil && tile.gameObjectIndex >= 5 {
				return false
			}
		}
	}
	gameObject := NewGameObject()
	gameObject.uid = uid
	gameObject.mask = objectRotationType
	gameObject.z = zLoc
	gameObject.x = xPos
	gameObject.y = yPos
	gameObject.tileHeight = tileHeight
	gameObject.renderable = renderable
	gameObject.turnValue = turnValue
	gameObject.xLocLow = xLoc
	gameObject.yLocHigh = yLoc
	gameObject.xLocHigh = xLoc + sizeX - 1
	gameObject.yLocLow = yLoc + sizeY - 1
	for x := xLoc; x < xLoc+sizeX; x++ {
		for y := yLoc; y < yLoc+sizeY; y++ {
			mask := 0
			if x > xLoc {
				mask++
			}
			if x < xLoc+sizeX-1 {
				mask += 4
			}
			if y > yLoc {
				mask += 8
			}
			if y < yLoc+sizeY-1 {
				mask += 2
			}
			for z := zLoc; z >= 0; z-- {
				if tileArray[z][x][y] == nil {
					tileArray[z][x][y] = NewTile(z, x, y)
				}
			}
			tile := tileArray[zLoc][x][y]
			tile[tile.gameObjectIndex] = gameObject
			tile[tile.gameObjectIndex] = mask
			tile.totalTiledObjectMask |= mask
			tile.gameObjectIndex++
		}
	}
	if isDynamic {
		gameObjectCache[++rcvr.interactableObjectCacheCurrPos] = gameObject
	}
	return true
}
func (rcvr *SceneGraph) AddGroundDecoration(zLoc int, zPos int, yLoc int, renderable *Renderable, objectRotationType byte, uid int, xLoc int) {
	if renderable == nil {
		return
	}
	groundDecoration := NewGroundDecoration()
	groundDecoration.renderable = renderable
	groundDecoration.xPos = xLoc*128 + 64
	groundDecoration.yPos = yLoc*128 + 64
	groundDecoration.zPos = zPos
	groundDecoration.uid = uid
	groundDecoration.mask = objectRotationType
	if tileArray[zLoc][xLoc][yLoc] == nil {
		tileArray[zLoc][xLoc][yLoc] = NewTile(zLoc, xLoc, yLoc)
	}
	<<unimp_obj.nm_*parser.GoArrayReference>> = groundDecoration
}
func (rcvr *SceneGraph) AddTile(zLoc int, xLoc int, yLoc int, shape int, i1 int, j1 int, k1 int, l1 int, i2 int, j2 int, k2 int, l2 int, i3 int, j3 int, k3 int, l3 int, i4 int, j4 int, k4 int, l4 int) {
	if shape == 0 {
		simpleTile := NewSimpleTile(k2, l2, i3, j3, -1, k4, false)
		for lowerZLoc := zLoc; lowerZLoc >= 0; lowerZLoc-- {
			if tileArray[lowerZLoc][xLoc][yLoc] == nil {
				tileArray[lowerZLoc][xLoc][yLoc] = NewTile(lowerZLoc, xLoc, yLoc)
			}
		}
		<<unimp_obj.nm_*parser.GoArrayReference>> = simpleTile
	} else if shape == 1 {
		simpleTile := NewSimpleTile(k3, l3, i4, j4, j1, l4, k1 == l1 && k1 == i2 && k1 == j2)
		for lowerZLoc := zLoc; lowerZLoc >= 0; lowerZLoc-- {
			if tileArray[lowerZLoc][xLoc][yLoc] == nil {
				tileArray[lowerZLoc][xLoc][yLoc] = NewTile(lowerZLoc, xLoc, yLoc)
			}
		}
		<<unimp_obj.nm_*parser.GoArrayReference>> = simpleTile
	} else {
		shapedTile := NewShapedTile(yLoc, k3, j3, i2, j1, i4, i1, k2, k4, i3, j2, l1, k1, shape, j4, l3, l2, xLoc, l4)
		for k5 := zLoc; k5 >= 0; k5-- {
			if tileArray[k5][xLoc][yLoc] == nil {
				tileArray[k5][xLoc][yLoc] = NewTile(k5, xLoc, yLoc)
			}
		}
		<<unimp_obj.nm_*parser.GoArrayReference>> = shapedTile
	}
}
func (rcvr *SceneGraph) AddTiledObject(uid int, objectRotationType byte, tileHeight int, sizeY int, renderable *Renderable, sizeX int, zLoc int, turnValue int, yLoc int, xLoc int) (bool) {
	if renderable == nil {
		return true
	} else {
		xPos := xLoc*128 + 64*sizeX
		yPos := yLoc*128 + 64*sizeY
		return rcvr.addAnimableC(zLoc, xLoc, yLoc, sizeX, sizeY, xPos, yPos, tileHeight, renderable, turnValue, false, uid, objectRotationType)
	}
}
func (rcvr *SceneGraph) AddWallDecoration(uid int, yLoc int, orientation2 int, zLoc int, xOffset int, zPos int, renderable *Renderable, xLoc int, objectRotationType byte, yOffset int, orientation int) {
	if renderable == nil {
		return
	}
	objectId := uint32(uid) >> 14 & 0x7fff
	wallDecoration := NewWallDecoration()
	wallDecoration.uid = uid
	wallDecoration.mask = objectRotationType
	wallDecoration.xPos = xLoc*128 + 64 + xOffset
	wallDecoration.yPos = yLoc*128 + 64 + yOffset
	wallDecoration.zPos = zPos
	wallDecoration.renderable = renderable
	wallDecoration.orientation = orientation
	wallDecoration.orientation2 = orientation2
	for z := zLoc; z >= 0; z-- {
		if tileArray[z][xLoc][yLoc] == nil {
			tileArray[z][xLoc][yLoc] = NewTile(z, xLoc, yLoc)
		}
	}
	<<unimp_obj.nm_*parser.GoArrayReference>> = wallDecoration
}
func (rcvr *SceneGraph) AddWallObject(orientation1 int, renderable1 *Renderable, uid int, yLoc int, objectFaceType byte, xLoc int, renderable2 *Renderable, zPos int, orientation2 int, zLoc int) {
	if renderable1 == nil && renderable2 == nil {
		return
	}
	wallObject := NewWallObject()
	wallObject.uid = uid
	wallObject.mask = objectFaceType
	wallObject.xPos = xLoc*128 + 64
	wallObject.yPos = yLoc*128 + 64
	wallObject.zPos = zPos
	wallObject.renderable1 = renderable1
	wallObject.renderable2 = renderable2
	wallObject.orientation1 = orientation1
	wallObject.orientation2 = orientation2
	for z := zLoc; z >= 0; z-- {
		if tileArray[z][xLoc][yLoc] == nil {
			tileArray[z][xLoc][yLoc] = NewTile(z, xLoc, yLoc)
		}
	}
	<<unimp_obj.nm_*parser.GoArrayReference>> = wallObject
}
func (rcvr *SceneGraph) ApplyBridgeMode(yLoc int, xLoc int) {
	tileFirstFloor := tileArray[0][xLoc][yLoc]
	for zLoc := 0; zLoc < 3; zLoc++ {
		tile := tileArray[zLoc][xLoc][yLoc] = tileArray[zLoc+1][xLoc][yLoc]
		if tile != nil {
			tile.z--
			for j1 := 0; j1 < tile.gameObjectIndex; j1++ {
				gameObject := tile[j1]
				if uint32(gameObject.uid)>>29&3 == 2 && gameObject.xLocLow == xLoc && gameObject.yLocHigh == yLoc {
					gameObject.z--
				}
			}
		}
	}
	if tileArray[0][xLoc][yLoc] == nil {
		tileArray[0][xLoc][yLoc] = NewTile(0, xLoc, yLoc)
	}
	<<unimp_obj.nm_*parser.GoArrayReference>> = tileFirstFloor
	tileArray[3][xLoc][yLoc] = nil
}
func (rcvr *SceneGraph) ClearGameObjectCache() {
	for i := 0; i < rcvr.interactableObjectCacheCurrPos; i++ {
		object5 := gameObjectCache[i]
		rcvr.remove(object5)
		gameObjectCache[i] = nil
	}
	rcvr.interactableObjectCacheCurrPos = 0
}
func CreateNewSceneCluster(z int, lowestX int, lowestZ int, highestX int, highestY int, highestZ int, lowestY int, searchMask int) {
	sceneCluster := NewSceneCluster()
	sceneCluster.startXLoc = lowestX / 128
	sceneCluster.endXLoc = highestX / 128
	sceneCluster.startYLoc = lowestY / 128
	sceneCluster.endYLoc = highestY / 128
	sceneCluster.orientation = searchMask
	sceneCluster.startXPos = lowestX
	sceneCluster.endXPos = highestX
	sceneCluster.startYPos = lowestY
	sceneCluster.endYPos = highestY
	sceneCluster.startZPos = highestZ
	sceneCluster.endZPos = lowestZ
	sceneClusters[z][++sceneClusterCounts[z]] = sceneCluster
}
func (rcvr *SceneGraph) GetWallObjectUid(zLoc int, xLoc int, yLoc int) (int) {
	tile := tileArray[zLoc][xLoc][yLoc]
	if tile == nil || tile.wallObject == nil {
		return 0
	} else {
		return tile.wallObject.uid
	}
}
func (rcvr *SceneGraph) InitializeToNull() {
	for zLoc := 0; zLoc < rcvr.zRegionSize; zLoc++ {
		for xLoc := 0; xLoc < rcvr.xRegionSize; xLoc++ {
			for yLoc := 0; yLoc < rcvr.yRegionSize; yLoc++ {
				tileArray[zLoc][xLoc][yLoc] = nil
			}
		}
	}
	for plane := 0; plane < cullingClusterPlaneCount; plane++ {
		for index := 0; index < sceneClusterCounts[plane]; index++ {
			sceneClusters[plane][index] = nil
		}
		sceneClusterCounts[plane] = 0
	}
	for index := 0; index < rcvr.interactableObjectCacheCurrPos; index++ {
		gameObjectCache[index] = nil
	}
	rcvr.interactableObjectCacheCurrPos = 0
	for index := 0; index < len(interactableObjects); index++ {
		interactableObjects[index] = nil
	}
}
func (rcvr *SceneGraph) light(j int, k int) (int) {
	k = 127 - k
	k = k * (j & 0x7f) / 160
	if k < 2 {
		k = 2
	} else if k > 126 {
		k = 126
	}
	return j&0xff80 + k
}
func (rcvr *SceneGraph) mergeNormals(model1 *Model, model2 *Model, offsetX int, offsetY int, offsetZ int, flag bool) {
	rcvr.mergeNormalsIndex++
	count := 0
	second := model2.vertexX
	secondVertices := model2.verticeCount
	for model1Vertex := 0; model1Vertex < model1.verticeCount; model1Vertex++ {
		vertexNormal1 := model1[model1Vertex]
		alsoVertexNormal1 := model1[model1Vertex]
		if alsoVertexNormal1.magnitude != 0 {
			dY := model1[model1Vertex] - offsetY
			if dY <= model2.maximumYVertex {
				dX := model1[model1Vertex] - offsetX
				if dX >= model2.minimumXVertex && dX <= model2.maximumXVertex {
					k2 := model1[model1Vertex] - offsetZ
					if k2 >= model2.minimumZVertex && k2 <= model2.maximumZVertex {
						for l2 := 0; l2 < secondVertices; l2++ {
							vertexNormal2 := model2[l2]
							alsoVertexNormal2 := model2[l2]
							if dX == second[l2] && k2 == model2[l2] && dY == model2[l2] && alsoVertexNormal2.magnitude != 0 {
								vertexNormal1.normalX += alsoVertexNormal2.normalX
								vertexNormal1.normalY += alsoVertexNormal2.normalY
								vertexNormal1.normalZ += alsoVertexNormal2.normalZ
								vertexNormal1.magnitude += alsoVertexNormal2.magnitude
								vertexNormal2.normalX += alsoVertexNormal1.normalX
								vertexNormal2.normalY += alsoVertexNormal1.normalY
								vertexNormal2.normalZ += alsoVertexNormal1.normalZ
								vertexNormal2.magnitude += alsoVertexNormal1.magnitude
								count++
								mergeANormals[model1Vertex] = rcvr.mergeNormalsIndex
								mergeBNormals[l2] = rcvr.mergeNormalsIndex
							}
						}
					}
				}
			}
		}
	}
	if count < 3 || !flag {
		return
	}
	for k1 := 0; k1 < model1.triangleCount; k1++ {
		if mergeANormals[model1[k1]] == rcvr.mergeNormalsIndex && mergeANormals[model1[k1]] == rcvr.mergeNormalsIndex && mergeANormals[model1[k1]] == rcvr.mergeNormalsIndex {
			model1[k1] = -1
		}
	}
	for l1 := 0; l1 < model2.triangleCount; l1++ {
		if mergeBNormals[model2[l1]] == rcvr.mergeNormalsIndex && mergeBNormals[model2[l1]] == rcvr.mergeNormalsIndex && mergeBNormals[model2[l1]] == rcvr.mergeNormalsIndex {
			model2[l1] = -1
		}
	}
}
func (rcvr *SceneGraph) Method275(zLoc int) {
	rcvr.cameraLowTileZ = zLoc
	for xLoc := 0; xLoc < rcvr.xRegionSize; xLoc++ {
		for yLoc := 0; yLoc < rcvr.yRegionSize; yLoc++ {
			if tileArray[zLoc][xLoc][yLoc] == nil {
				tileArray[zLoc][xLoc][yLoc] = NewTile(zLoc, xLoc, yLoc)
			}
		}
	}
}
func (rcvr *SceneGraph) Method290(yLoc int, k int, xLoc int, zLoc int) {
	tile := tileArray[zLoc][xLoc][yLoc]
	if tile == nil {
		return
	}
	wallDecoration := tile.wallDecoration
	if wallDecoration != nil {
		xPos := xLoc*128 + 64
		yPos := yLoc*128 + 64
		wallDecoration.xPos = xPos + (wallDecoration.xPos-xPos)*k/16
		wallDecoration.yPos = yPos + (wallDecoration.yPos-yPos)*k/16
	}
}
func (rcvr *SceneGraph) method306GroundDecorationOnly(modelXLoc int, modelZLoc int, model *Model, modelYLoc int) {
	if modelXLoc < rcvr.xRegionSize {
		tile := tileArray[modelZLoc][modelXLoc+1][modelYLoc]
		if tile != nil && tile.groundDecoration != nil && tile.groundDecoration.renderable.vertexNormals != nil {
			rcvr.mergeNormals(model, tile.groundDecoration.renderable.(*Model), 128, 0, 0, true)
		}
	}
	if modelYLoc < rcvr.xRegionSize {
		tile := tileArray[modelZLoc][modelXLoc][modelYLoc+1]
		if tile != nil && tile.groundDecoration != nil && tile.groundDecoration.renderable.vertexNormals != nil {
			rcvr.mergeNormals(model, tile.groundDecoration.renderable.(*Model), 0, 0, 128, true)
		}
	}
	if modelXLoc < rcvr.xRegionSize && modelYLoc < rcvr.yRegionSize {
		tile := tileArray[modelZLoc][modelXLoc+1][modelYLoc+1]
		if tile != nil && tile.groundDecoration != nil && tile.groundDecoration.renderable.vertexNormals != nil {
			rcvr.mergeNormals(model, tile.groundDecoration.renderable.(*Model), 128, 0, 128, true)
		}
	}
	if modelXLoc < rcvr.xRegionSize && modelYLoc > 0 {
		tile := tileArray[modelZLoc][modelXLoc+1][modelYLoc-1]
		if tile != nil && tile.groundDecoration != nil && tile.groundDecoration.renderable.vertexNormals != nil {
			rcvr.mergeNormals(model, tile.groundDecoration.renderable.(*Model), 128, 0, -128, true)
		}
	}
}
func (rcvr *SceneGraph) method307(modelZLoc int, modelXSize int, modelYSize int, modelXLoc int, modelYLoc int, model *Model) {
	flag := true
	startX := modelXLoc
	stopX := modelXLoc + modelXSize
	startY := modelYLoc - 1
	stopY := modelYLoc + modelYSize
	for zLoc := modelZLoc; zLoc <= modelZLoc+1; zLoc++ {
		if zLoc != rcvr.zRegionSize {
			for xLoc := startX; xLoc <= stopX; xLoc++ {
				if xLoc >= 0 && xLoc < rcvr.xRegionSize {
					for yLoc := startY; yLoc <= stopY; yLoc++ {
						if yLoc >= 0 && yLoc < rcvr.yRegionSize && (!flag || xLoc >= stopX || yLoc >= stopY || yLoc < modelYLoc && xLoc != modelXLoc) {
							tile := tileArray[zLoc][xLoc][yLoc]
							if tile != nil {
								relativeHeightToModelTile := (heightMap[zLoc][xLoc][yLoc]+heightMap[zLoc][xLoc+1][yLoc]+heightMap[zLoc][xLoc][yLoc+1]+heightMap[zLoc][xLoc+1][yLoc+1])/4 - (heightMap[modelZLoc][modelXLoc][modelYLoc]+heightMap[modelZLoc][modelXLoc+1][modelYLoc]+heightMap[modelZLoc][modelXLoc][modelYLoc+1]+heightMap[modelZLoc][modelXLoc+1][modelYLoc+1])/4
								wallObject := tile.wallObject
								if wallObject != nil && wallObject.renderable1 != nil && wallObject.renderable1.vertexNormals != nil {
									rcvr.mergeNormals(model, wallObject.renderable1.(*Model), (xLoc-modelXLoc)*128+(1-modelXSize)*64, relativeHeightToModelTile, (yLoc-modelYLoc)*128+(1-modelYSize)*64, flag)
								}
								if wallObject != nil && wallObject.renderable2 != nil && wallObject.renderable2.vertexNormals != nil {
									rcvr.mergeNormals(model, wallObject.renderable2.(*Model), (xLoc-modelXLoc)*128+(1-modelXSize)*64, relativeHeightToModelTile, (yLoc-modelYLoc)*128+(1-modelYSize)*64, flag)
								}
								for i := 0; i < tile.gameObjectIndex; i++ {
									gameObject := tile[i]
									if gameObject != nil && gameObject.renderable != nil && gameObject.renderable.vertexNormals != nil {
										tiledObjectXSize := gameObject.xLocHigh - gameObject.xLocLow + 1
										tiledObjectYSize := gameObject.yLocLow - gameObject.yLocHigh + 1
										mergeNormals(model, gameObject.renderable.(*Model), (gameObject.xLocLow-modelXLoc)*128+(tiledObjectXSize-modelXSize)*64, relativeHeightToModelTile, (gameObject.yLocHigh-modelYLoc)*128+(tiledObjectYSize-modelYSize)*64, flag)
									}
								}
							}
						}
					}
				}
			}
			startX--
			flag = false
		}
	}
}
func method311(i int, j int, k int) (bool) {
	l := uint32(j*camLeftRightY+k*camLeftRightX) >> 16
	i1 := uint32(j*camLeftRightX-k*camLeftRightY) >> 16
	j1 := uint32(i*camUpDownY+i1*camUpDownX) >> 16
	k1 := uint32(i*camUpDownX-i1*camUpDownY) >> 16
	if j1 < 50 || j1 > 3500 {
		return false
	}
	l1 := viewportHalfWidth + l<<ViewDistance/j1
	i2 := viewportHalfHeight + k1<<ViewDistance/j1
	return l1 >= anInt495 && l1 <= viewportWidth && i2 >= anInt496 && i2 <= viewportHeight
}
func (rcvr *SceneGraph) method315(simpleTile *SimpleTile, i int, j int, k int, l int, i1 int, j1 int, k1 int) {
	var l1 int
	i2 := l1 = j1<<7-xCameraPos
	var j2 int
	k2 := j2 = k1<<7-yCameraPos
	var l2 int
	i3 := l2 = i2+128
	var j3 int
	k3 := j3 = k2+128
	l3 := heightMap[i][j1][k1] - zCameraPos
	i4 := heightMap[i][j1+1][k1] - zCameraPos
	j4 := heightMap[i][j1+1][k1+1] - zCameraPos
	k4 := heightMap[i][j1][k1+1] - zCameraPos
	l4 := uint32(k2*l+i2*i1) >> 16
	k2 = uint32(k2*i1-i2*l) >> 16
	i2 = l4
	l4 = uint32(l3*k-k2*j) >> 16
	k2 = uint32(l3*j+k2*k) >> 16
	l3 = l4
	if k2 < 50 {
		return
	}
	l4 = uint32(j2*l+i3*i1) >> 16
	j2 = uint32(j2*i1-i3*l) >> 16
	i3 = l4
	l4 = uint32(i4*k-j2*j) >> 16
	j2 = uint32(i4*j+j2*k) >> 16
	i4 = l4
	if j2 < 50 {
		return
	}
	l4 = uint32(k3*l+l2*i1) >> 16
	k3 = uint32(k3*i1-l2*l) >> 16
	l2 = l4
	l4 = uint32(j4*k-k3*j) >> 16
	k3 = uint32(j4*j+k3*k) >> 16
	j4 = l4
	if k3 < 50 {
		return
	}
	l4 = uint32(j3*l+l1*i1) >> 16
	j3 = uint32(j3*i1-l1*l) >> 16
	l1 = l4
	l4 = uint32(k4*k-j3*j) >> 16
	j3 = uint32(k4*j+j3*k) >> 16
	k4 = l4
	if j3 < 50 {
		return
	}
	i5 := Rasterizer3D.originViewX + i2<<ViewDistance/k2
	j5 := Rasterizer3D.originViewY + l3<<ViewDistance/k2
	k5 := Rasterizer3D.originViewX + i3<<ViewDistance/j2
	l5 := Rasterizer3D.originViewY + i4<<ViewDistance/j2
	i6 := Rasterizer3D.originViewX + l2<<ViewDistance/k3
	j6 := Rasterizer3D.originViewY + j4<<ViewDistance/k3
	k6 := Rasterizer3D.originViewX + l1<<ViewDistance/j3
	l6 := Rasterizer3D.originViewY + k4<<ViewDistance/j3
	Rasterizer3D.alpha = 0
	if (i6-k6)*(l5-l6)-(j6-l6)*(k5-k6) > 0 {
		Rasterizer3D.textureOutOfDrawingBounds = i6 < 0 || k6 < 0 || k5 < 0 || i6 > Rasterizer2D.lastX || k6 > Rasterizer2D.lastX || k5 > Rasterizer2D.lastX
		if clicked && rcvr.method318(clickScreenX, clickScreenY, j6, l6, l5, i6, k6, k5) {
			clickedTileX = j1
			clickedTileY = k1
		}
		if simpleTile.getTexture() == -1 {
			if simpleTile.getCenterColor() != 0xbc614e {
				Rasterizer3D.drawShadedTriangle(j6, l6, l5, i6, k6, k5, simpleTile.getCenterColor(), simpleTile.getEastColor(), simpleTile.getNorthColor(), k3, j3, j2)
			}
		} else {
			textureColor := tEXTURE_COLORS[simpleTile.getTexture()]
			Rasterizer3D.drawShadedTriangle(j6, l6, l5, i6, k6, k5, rcvr.light(textureColor, simpleTile.getCenterColor()), light(textureColor, simpleTile.getEastColor()), light(textureColor, simpleTile.getNorthColor()), k3, j3, j2)
		}
	}
	if (i5-k5)*(l6-l5)-(j5-l5)*(k6-k5) > 0 {
		Rasterizer3D.textureOutOfDrawingBounds = i5 < 0 || k5 < 0 || k6 < 0 || i5 > Rasterizer2D.lastX || k5 > Rasterizer2D.lastX || k6 > Rasterizer2D.lastX
		if clicked && rcvr.method318(clickScreenX, clickScreenY, j5, l5, l6, i5, k5, k6) {
			clickedTileX = j1
			clickedTileY = k1
		}
		if simpleTile.getTexture() == -1 {
			if simpleTile.getNorthEastColor() != 0xbc614e {
				Rasterizer3D.drawShadedTriangle(j5, l5, l6, i5, k5, k6, simpleTile.getNorthEastColor(), simpleTile.getNorthColor(), simpleTile.getEastColor(), k2, j2, j3)
			}
		} else {
			j7 := tEXTURE_COLORS[simpleTile.getTexture()]
			Rasterizer3D.drawShadedTriangle(j5, l5, l6, i5, k5, k6, light(j7, simpleTile.getNorthEastColor()), light(j7, simpleTile.getNorthColor()), light(j7, simpleTile.getEastColor()), k2, j2, j3)
		}
	}
}
func (rcvr *SceneGraph) method316(i int, j int, k int, class40 *ShapedTile, l int, i1 int, j1 int) {
	k1 := class40.anIntArray673.length
	for l1 := 0; l1 < k1; l1++ {
		i2 := class40[l1] - xCameraPos
		k2 := class40[l1] - zCameraPos
		i3 := class40[l1] - yCameraPos
		k3 := uint32(i3*k+i2*j1) >> 16
		i3 = uint32(i3*j1-i2*k) >> 16
		i2 = k3
		k3 = uint32(k2*l-i3*j) >> 16
		i3 = uint32(k2*j+i3*l) >> 16
		k2 = k3
		if i3 < 50 {
			return
		}
		if class40.anIntArray682 != nil {
			<<unimp_arrayref_ShapedTile.anIntArray690>>[l1] = i2
			<<unimp_arrayref_ShapedTile.anIntArray691>>[l1] = k2
			<<unimp_arrayref_ShapedTile.anIntArray692>>[l1] = i3
		}
		<<unimp_arrayref_ShapedTile.anIntArray688>>[l1] = Rasterizer3D.originViewX + i2<<ViewDistance/i3
		<<unimp_arrayref_ShapedTile.anIntArray689>>[l1] = Rasterizer3D.originViewY + k2<<ViewDistance/i3
	}
	Rasterizer3D.alpha = 0
	k1 = class40.anIntArray679.length
	for j2 := 0; j2 < k1; j2++ {
		l2 := class40[j2]
		j3 := class40[j2]
		l3 := class40[j2]
		i4 := <<unimp_arrayref_ShapedTile.anIntArray688>>[l2]
		j4 := <<unimp_arrayref_ShapedTile.anIntArray688>>[j3]
		k4 := <<unimp_arrayref_ShapedTile.anIntArray688>>[l3]
		l4 := <<unimp_arrayref_ShapedTile.anIntArray689>>[l2]
		i5 := <<unimp_arrayref_ShapedTile.anIntArray689>>[j3]
		j5 := <<unimp_arrayref_ShapedTile.anIntArray689>>[l3]
		if (i4-j4)*(j5-i5)-(l4-i5)*(k4-j4) > 0 {
			Rasterizer3D.textureOutOfDrawingBounds = i4 < 0 || j4 < 0 || k4 < 0 || i4 > Rasterizer2D.lastX || j4 > Rasterizer2D.lastX || k4 > Rasterizer2D.lastX
			if clicked && rcvr.method318(clickScreenX, clickScreenY, l4, i5, j5, i4, j4, k4) {
				clickedTileX = i
				clickedTileY = i1
			}
			if class40.anIntArray682 == nil || class40[j2] == -1 {
				if class40[j2] != 0xbc614e {
					Rasterizer3D.drawShadedTriangle(l4, i5, j5, i4, j4, k4, class40[j2], class40[j2], class40[j2], <<unimp_arrayref_ShapedTile.depthPoint>>[l2], <<unimp_arrayref_ShapedTile.depthPoint>>[j3], <<unimp_arrayref_ShapedTile.depthPoint>>[l3])
				}
			} else {
				k5 := tEXTURE_COLORS[class40[j2]]
				Rasterizer3D.drawShadedTriangle(l4, i5, j5, i4, j4, k4, light(k5, class40[j2]), light(k5, class40[j2]), light(k5, class40[j2]), <<unimp_arrayref_ShapedTile.depthPoint>>[l2], <<unimp_arrayref_ShapedTile.depthPoint>>[j3], <<unimp_arrayref_ShapedTile.depthPoint>>[l3])
			}
		}
	}
}
func (rcvr *SceneGraph) method318(i int, j int, k int, l int, i1 int, j1 int, k1 int, l1 int) (bool) {
	if j < k && j < l && j < i1 {
		return false
	}
	if j > k && j > l && j > i1 {
		return false
	}
	if i < j1 && i < k1 && i < l1 {
		return false
	}
	if i > j1 && i > k1 && i > l1 {
		return false
	}
	i2 := (j-k)*(k1-j1) - (i-j1)*(l-k)
	j2 := (j-i1)*(j1-l1) - (i-l1)*(k-i1)
	k2 := (j-l)*(l1-k1) - (i-k1)*(i1-l)
	return i2*k2 > 0 && k2*j2 > 0
}
func (rcvr *SceneGraph) method321(i int, j int, k int, l int) (bool) {
	if !rcvr.tileVisible(i, j, k) {
		return false
	}
	i1 := j << 7
	j1 := k << 7
	k1 := heightMap[i][j][k] - 1
	l1 := k1 - 120
	i2 := k1 - 230
	j2 := k1 - 238
	if l < 16 {
		if l == 1 {
			if i1 > xCameraPos {
				if !visible(i1, k1, j1) {
					return false
				}
				if !visible(i1, k1, j1+128) {
					return false
				}
			}
			if i > 0 {
				if !visible(i1, l1, j1) {
					return false
				}
				if !visible(i1, l1, j1+128) {
					return false
				}
			}
			return visible(i1, i2, j1) && visible(i1, i2, j1+128)
		}
		if l == 2 {
			if j1 < yCameraPos {
				if !visible(i1, k1, j1+128) {
					return false
				}
				if !visible(i1+128, k1, j1+128) {
					return false
				}
			}
			if i > 0 {
				if !visible(i1, l1, j1+128) {
					return false
				}
				if !visible(i1+128, l1, j1+128) {
					return false
				}
			}
			return visible(i1, i2, j1+128) && visible(i1+128, i2, j1+128)
		}
		if l == 4 {
			if i1 < xCameraPos {
				if !visible(i1+128, k1, j1) {
					return false
				}
				if !visible(i1+128, k1, j1+128) {
					return false
				}
			}
			if i > 0 {
				if !visible(i1+128, l1, j1) {
					return false
				}
				if !visible(i1+128, l1, j1+128) {
					return false
				}
			}
			return visible(i1+128, i2, j1) && visible(i1+128, i2, j1+128)
		}
		if l == 8 {
			if j1 > yCameraPos {
				if !visible(i1, k1, j1) {
					return false
				}
				if !visible(i1+128, k1, j1) {
					return false
				}
			}
			if i > 0 {
				if !visible(i1, l1, j1) {
					return false
				}
				if !visible(i1+128, l1, j1) {
					return false
				}
			}
			return visible(i1, i2, j1) && visible(i1+128, i2, j1)
		}
	}
	if !visible(i1+64, j2, j1+64) {
		return false
	}
	if l == 16 {
		return visible(i1, i2, j1+128)
	}
	if l == 32 {
		return visible(i1+128, i2, j1+128)
	}
	if l == 64 {
		return visible(i1+128, i2, j1)
	}
	if l == 128 {
		return visible(i1, i2, j1)
	} else {
		fmt.Println("Warning unsupported wall type")
		return true
	}
}
func (rcvr *SceneGraph) method322(i int, j int, k int, l int) (bool) {
	if !rcvr.tileVisible(i, j, k) {
		return false
	}
	i1 := j << 7
	j1 := k << 7
	return visible(i1+1, heightMap[i][j][k]-l, j1+1) && visible(i1+128-1, heightMap[i][j+1][k]-l, j1+1) && visible(i1+128-1, heightMap[i][j+1][k+1]-l, j1+128-1) && visible(i1+1, heightMap[i][j][k+1]-l, j1+128-1)
}
func (rcvr *SceneGraph) method323(i int, j int, k int, l int, i1 int, j1 int) (bool) {
	if j == k && l == i1 {
		if !rcvr.tileVisible(i, j, l) {
			return false
		}
		k1 := j << 7
		i2 := l << 7
		return visible(k1+1, heightMap[i][j][l]-j1, i2+1) && visible(k1+128-1, heightMap[i][j+1][l]-j1, i2+1) && visible(k1+128-1, heightMap[i][j+1][l+1]-j1, i2+128-1) && visible(k1+1, heightMap[i][j][l+1]-j1, i2+128-1)
	}
	for l1 := j; l1 <= k; l1++ {
		for j2 := l; j2 <= i1; j2++ {
			if renderedViewableObjects[i][l1][j2] == -renderedObjectCount {
				return false
			}
		}
	}
	k2 := j<<7 + 1
	l2 := l<<7 + 2
	i3 := heightMap[i][j][l] - j1
	if !visible(k2, i3, l2) {
		return false
	}
	j3 := k<<7 - 1
	if !visible(j3, i3, l2) {
		return false
	}
	k3 := i1<<7 - 1
	return visible(k2, i3, k3) && visible(j3, i3, k3)
}
func (rcvr *SceneGraph) processCulling() {
	sceneClusterCount := sceneClusterCounts[currentRenderPlane]
	sceneClusters := <<unimp_arrayref_SceneGraph.sceneClusters>>[currentRenderPlane]
	processedCullingCluster = 0
	for sceneIndex := 0; sceneIndex < sceneClusterCount; sceneIndex++ {
		sceneCluster := sceneClusters[sceneIndex]
		if sceneCluster.orientation == 1 {
			relativeX := sceneCluster.startXLoc - xCameraTile + 25
			if relativeX < 0 || relativeX > 50 {
				continue
			}
			minRelativeY := sceneCluster.startYLoc - yCameraTile + 25
			if minRelativeY < 0 {
				minRelativeY = 0
			}
			maxRelativeY := sceneCluster.endYLoc - yCameraTile + 25
			if maxRelativeY > 50 {
				maxRelativeY = 50
			}
			visible := false
			if !visible {
				continue
			}
			dXPos := xCameraPos - sceneCluster.startXPos
			if dXPos > 32 {
				sceneCluster.cullDirection = 1
			} else {
				if dXPos >= -32 {
					continue
				}
				sceneCluster.cullDirection = 2
				dXPos = -dXPos
			}
			sceneCluster.cameraDistanceStartY = (sceneCluster.startYPos - yCameraPos) << 8 / dXPos
			sceneCluster.cameraDistanceEndY = (sceneCluster.endYPos - yCameraPos) << 8 / dXPos
			sceneCluster.cameraDistanceStartZ = (sceneCluster.startZPos - zCameraPos) << 8 / dXPos
			sceneCluster.cameraDistanceEndZ = (sceneCluster.endZPos - zCameraPos) << 8 / dXPos
			fixedCullingClusters[++processedCullingCluster] = sceneCluster
			continue
		}
		if sceneCluster.orientation == 2 {
			relativeY := sceneCluster.startYLoc - yCameraTile + 25
			if relativeY < 0 || relativeY > 50 {
				continue
			}
			minRelativeX := sceneCluster.startXLoc - xCameraTile + 25
			if minRelativeX < 0 {
				minRelativeX = 0
			}
			maxRelativeX := sceneCluster.endXLoc - xCameraTile + 25
			if maxRelativeX > 50 {
				maxRelativeX = 50
			}
			visible := false
			if !visible {
				continue
			}
			dYPos := yCameraPos - sceneCluster.startYPos
			if dYPos > 32 {
				sceneCluster.cullDirection = 3
			} else if dYPos < -32 {
				sceneCluster.cullDirection = 4
				dYPos = -dYPos
			} else {
				continue
			}
			sceneCluster.cameraDistanceStartX = (sceneCluster.startXPos - xCameraPos) << 8 / dYPos
			sceneCluster.cameraDistanceEndX = (sceneCluster.endXPos - xCameraPos) << 8 / dYPos
			sceneCluster.cameraDistanceStartZ = (sceneCluster.startZPos - zCameraPos) << 8 / dYPos
			sceneCluster.cameraDistanceEndZ = (sceneCluster.endZPos - zCameraPos) << 8 / dYPos
			fixedCullingClusters[++processedCullingCluster] = sceneCluster
		} else if sceneCluster.orientation == 4 {
			relativeZ := sceneCluster.startZPos - zCameraPos
			if relativeZ > 128 {
				minRelativeY := sceneCluster.startYLoc - yCameraTile + 25
				if minRelativeY < 0 {
					minRelativeY = 0
				}
				maxRelativeY := sceneCluster.endYLoc - yCameraTile + 25
				if maxRelativeY > 50 {
					maxRelativeY = 50
				}
				if minRelativeY <= maxRelativeY {
					minRelativeX := sceneCluster.startXLoc - xCameraTile + 25
					if minRelativeX < 0 {
						minRelativeX = 0
					}
					maxRelativeX := sceneCluster.endXLoc - xCameraTile + 25
					if maxRelativeX > 50 {
						maxRelativeX = 50
					}
					visible := false
					if visible {
						sceneCluster.cullDirection = 5
						sceneCluster.cameraDistanceStartX = (sceneCluster.startXPos - xCameraPos) << 8 / relativeZ
						sceneCluster.cameraDistanceEndX = (sceneCluster.endXPos - xCameraPos) << 8 / relativeZ
						sceneCluster.cameraDistanceStartY = (sceneCluster.startYPos - yCameraPos) << 8 / relativeZ
						sceneCluster.cameraDistanceEndY = (sceneCluster.endYPos - yCameraPos) << 8 / relativeZ
						fixedCullingClusters[++processedCullingCluster] = sceneCluster
					}
				}
			}
		}
	}
}
func (rcvr *SceneGraph) remove(gameObject *GameObject) {
	for x := gameObject.xLocLow; x <= gameObject.xLocHigh; x++ {
		for y := gameObject.yLocHigh; y <= gameObject.yLocLow; y++ {
			tile := tileArray[gameObject.z][x][y]
			if tile != nil {
				for i := 0; i < tile.gameObjectIndex; i++ {
					if tile[i] != gameObject {
						continue
					}
					tile.gameObjectIndex--
					for i1 := i; i1 < tile.gameObjectIndex; i1++ {
						tile[i1] = tile[i1+1]
						tile[i1] = tile[i1+1]
					}
					tile[tile.gameObjectIndex] = nil
					break
				}
				tile.totalTiledObjectMask = 0
				for i := 0; i < tile.gameObjectIndex; i++ {
					tile.totalTiledObjectMask |= tile[i]
				}
			}
		}
	}
}
func (rcvr *SceneGraph) Render(cameraXPos int, cameraYPos int, camAngleXY int, cameraZPos int, planeZ int, camAngleZ int) {
	if cameraXPos < 0 {
		cameraXPos = 0
	} else if cameraXPos >= rcvr.xRegionSize*128 {
		cameraXPos = rcvr.xRegionSize*128 - 1
	}
	if cameraYPos < 0 {
		cameraYPos = 0
	} else if cameraYPos >= rcvr.yRegionSize*128 {
		cameraYPos = rcvr.yRegionSize*128 - 1
	}
	renderedObjectCount++
	camUpDownY = <<unimp_arrayref_Model.sine>>[camAngleZ]
	camUpDownX = <<unimp_arrayref_Model.cosine>>[camAngleZ]
	camLeftRightY = <<unimp_arrayref_Model.sine>>[camAngleXY]
	camLeftRightX = <<unimp_arrayref_Model.cosine>>[camAngleXY]
	xCameraPos = cameraXPos
	zCameraPos = cameraZPos
	yCameraPos = cameraYPos
	xCameraTile = cameraXPos / 128
	yCameraTile = cameraYPos / 128
	currentRenderPlane = planeZ
	cameraLowTileX = xCameraTile - 25
	if cameraLowTileX < 0 {
		cameraLowTileX = 0
	}
	cameraLowTileY = yCameraTile - 25
	if cameraLowTileY < 0 {
		cameraLowTileY = 0
	}
	cameraHighTileX = xCameraTile + 25
	if cameraHighTileX > rcvr.xRegionSize {
		cameraHighTileX = rcvr.xRegionSize
	}
	cameraHighTileY = yCameraTile + 25
	if cameraHighTileY > rcvr.yRegionSize {
		cameraHighTileY = rcvr.yRegionSize
	}
	rcvr.processCulling()
	clickCount = 0
	for zLoc := rcvr.cameraLowTileZ; zLoc < rcvr.zRegionSize; zLoc++ {
		planeTiles := tileArray[zLoc]
		for xLoc := cameraLowTileX; xLoc < cameraHighTileX; xLoc++ {
			for yLoc := cameraLowTileY; yLoc < cameraHighTileY; yLoc++ {
				tile := planeTiles[xLoc][yLoc]
				if tile != nil {
					if tile.logicHeight > planeZ {
						tile.updated = false
						tile.drawn = false
						tile.renderMask = 0
					} else {
						tile.updated = true
						tile.drawn = true
						tile.multipleObjects = tile.gameObjectIndex > 0
						clickCount++
					}
				}
			}
		}
	}
	for zLoc := rcvr.cameraLowTileZ; zLoc < rcvr.zRegionSize; zLoc++ {
		plane := tileArray[zLoc]
		for dX := -25; dX <= 0; dX++ {
			xLocIncrement := xCameraTile + dX
			xLocDecrement := xCameraTile - dX
			if xLocIncrement >= cameraLowTileX || xLocDecrement < cameraHighTileX {
				for dY := -25; dY <= 0; dY++ {
					yLocIncrement := yCameraTile + dY
					yLocDecrement := yCameraTile - dY
					if xLocIncrement >= cameraLowTileX {
						if yLocIncrement >= cameraLowTileY {
							tile := plane[xLocIncrement][yLocIncrement]
							if tile != nil && tile.updated {
								rcvr.renderTile(tile, true)
							}
						}
						if yLocDecrement < cameraHighTileY {
							tile := plane[xLocIncrement][yLocDecrement]
							if tile != nil && tile.updated {
								rcvr.renderTile(tile, true)
							}
						}
					}
					if xLocDecrement < cameraHighTileX {
						if yLocIncrement >= cameraLowTileY {
							tile := plane[xLocDecrement][yLocIncrement]
							if tile != nil && tile.updated {
								rcvr.renderTile(tile, true)
							}
						}
						if yLocDecrement < cameraHighTileY {
							tile := plane[xLocDecrement][yLocDecrement]
							if tile != nil && tile.updated {
								rcvr.renderTile(tile, true)
							}
						}
					}
					if clickCount == 0 {
						clicked = false
						return
					}
				}
			}
		}
	}
	for zLoc := rcvr.cameraLowTileZ; zLoc < rcvr.zRegionSize; zLoc++ {
		plane := tileArray[zLoc]
		for dX := -25; dX <= 0; dX++ {
			xLocIncrement := xCameraTile + dX
			xLocDecrement := xCameraTile - dX
			if xLocIncrement >= cameraLowTileX || xLocDecrement < cameraHighTileX {
				for dY := -25; dY <= 0; dY++ {
					yLocIncrement := yCameraTile + dY
					yLocDecrement := yCameraTile - dY
					if xLocIncrement >= cameraLowTileX {
						if yLocIncrement >= cameraLowTileY {
							tile := plane[xLocIncrement][yLocIncrement]
							if tile != nil && tile.updated {
								rcvr.renderTile(tile, false)
							}
						}
						if yLocDecrement < cameraHighTileY {
							tile := plane[xLocIncrement][yLocDecrement]
							if tile != nil && tile.updated {
								rcvr.renderTile(tile, false)
							}
						}
					}
					if xLocDecrement < cameraHighTileX {
						if yLocIncrement >= cameraLowTileY {
							tile := plane[xLocDecrement][yLocIncrement]
							if tile != nil && tile.updated {
								rcvr.renderTile(tile, false)
							}
						}
						if yLocDecrement < cameraHighTileY {
							tile := plane[xLocDecrement][yLocDecrement]
							if tile != nil && tile.updated {
								rcvr.renderTile(tile, false)
							}
						}
					}
					if clickCount == 0 {
						clicked = false
						return
					}
				}
			}
		}
	}
	clicked = false
}
func (rcvr *SceneGraph) renderTile(renderTile *Tile, renderObjectOnWall bool) {
	tileDeque.insertHead(renderTile)
	for {
		var currentTile *Tile
		for {
			currentTile = tileDeque.popHead().(*Tile)
			if currentTile == nil {
				return
			}
			if !(!currentTile.drawn) {
				break
			}
		}
		x := currentTile.x
		y := currentTile.y
		z := currentTile.z
		plane := currentTile.plane
		tileHeights := tileArray[z]
		if currentTile.updated {
			if renderObjectOnWall {
				if z > 0 {
					tile := tileArray[z-1][x][y]
					if tile != nil && tile.drawn {
						continue
					}
				}
				if x <= xCameraTile && x > cameraLowTileX {
					tile := tileHeights[x-1][y]
					if tile != nil && tile.drawn && (tile.updated || currentTile.totalTiledObjectMask&1 == 0) {
						continue
					}
				}
				if x >= xCameraTile && x < cameraHighTileX-1 {
					tile := tileHeights[x+1][y]
					if tile != nil && tile.drawn && (tile.updated || currentTile.totalTiledObjectMask&4 == 0) {
						continue
					}
				}
				if y <= yCameraTile && y > cameraLowTileY {
					tile := tileHeights[x][y-1]
					if tile != nil && tile.drawn && (tile.updated || currentTile.totalTiledObjectMask&8 == 0) {
						continue
					}
				}
				if y >= yCameraTile && y < cameraHighTileY-1 {
					tile := tileHeights[x][y+1]
					if tile != nil && tile.drawn && (tile.updated || currentTile.totalTiledObjectMask&2 == 0) {
						continue
					}
				}
			} else {
				renderObjectOnWall = true
			}
			currentTile.updated = false
			if currentTile.firstFloorTile != nil {
				class30_sub3_7 := currentTile.firstFloorTile
				if class30_sub3_7.mySimpleTile != nil {
					if !rcvr.tileVisible(0, x, y) {
						rcvr.method315(class30_sub3_7.mySimpleTile, 0, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, x, y)
					}
				} else {
					if class30_sub3_7.myShapedTile != nil && !rcvr.tileVisible(0, x, y) {
						rcvr.method316(x, camUpDownY, camLeftRightY, class30_sub3_7.myShapedTile, camUpDownX, y, camLeftRightX)
					}
				}
				wall := class30_sub3_7.wallObject
				if wall != nil {
					wall.renderable1.renderAtPoint(0, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, wall.xPos-xCameraPos, wall.zPos-zCameraPos, wall.yPos-yCameraPos, wall.uid)
				}
				for index := 0; index < class30_sub3_7.gameObjectIndex; index++ {
					object := class30_sub3_7[index]
					if object != nil {
						object.renderable.renderAtPoint(object.turnValue, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, object.x-xCameraPos, object.tileHeight-zCameraPos, object.y-yCameraPos, object.uid)
					}
				}
			}
			renderDecoration := false
			if currentTile.mySimpleTile != nil {
				if !rcvr.tileVisible(plane, x, y) {
					renderDecoration = true
					rcvr.method315(currentTile.mySimpleTile, plane, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, x, y)
				}
			} else if currentTile.myShapedTile != nil && !rcvr.tileVisible(plane, x, y) {
				renderDecoration = true
				rcvr.method316(x, camUpDownY, camLeftRightY, currentTile.myShapedTile, camUpDownX, y, camLeftRightX)
			}
			j1 := 0
			j2 := 0
			class10_3 := currentTile.wallObject
			class26_1 := currentTile.wallDecoration
			if class10_3 != nil || class26_1 != nil {
				if xCameraTile == x {
					j1++
				} else if xCameraTile < x {
					j1 += 2
				}
				if yCameraTile == y {
					j1 += 3
				} else if yCameraTile > y {
					j1 += 6
				}
				j2 = anIntArray478[j1]
				currentTile.anInt1328 = anIntArray480[j1]
			}
			if class10_3 != nil {
				if class10_3.orientation1&anIntArray479[j1] != 0 {
					if class10_3.orientation1 == 16 {
						currentTile.renderMask = 3
						currentTile.anInt1326 = anIntArray481[j1]
						currentTile.anInt1327 = 3 - currentTile.anInt1326
					} else if class10_3.orientation1 == 32 {
						currentTile.renderMask = 6
						currentTile.anInt1326 = anIntArray482[j1]
						currentTile.anInt1327 = 6 - currentTile.anInt1326
					} else if class10_3.orientation1 == 64 {
						currentTile.renderMask = 12
						currentTile.anInt1326 = anIntArray483[j1]
						currentTile.anInt1327 = 12 - currentTile.anInt1326
					} else {
						currentTile.renderMask = 9
						currentTile.anInt1326 = anIntArray484[j1]
						currentTile.anInt1327 = 9 - currentTile.anInt1326
					}
				} else {
					currentTile.renderMask = 0
				}
				if class10_3.orientation1&j2 != 0 && !rcvr.method321(plane, x, y, class10_3.orientation1) {
					class10_3.renderable1.renderAtPoint(0, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class10_3.xPos-xCameraPos, class10_3.zPos-zCameraPos, class10_3.yPos-yCameraPos, class10_3.uid)
				}
				if class10_3.orientation2&j2 != 0 && !rcvr.method321(plane, x, y, class10_3.orientation2) {
					class10_3.renderable2.renderAtPoint(0, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class10_3.xPos-xCameraPos, class10_3.zPos-zCameraPos, class10_3.yPos-yCameraPos, class10_3.uid)
				}
			}
			if class26_1 != nil && !rcvr.method322(plane, x, y, class26_1.renderable.modelBaseY) {
				if class26_1.orientation&j2 != 0 {
					class26_1.renderable.renderAtPoint(class26_1.orientation2, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class26_1.xPos-xCameraPos, class26_1.zPos-zCameraPos, class26_1.yPos-yCameraPos, class26_1.uid)
				} else if class26_1.orientation&0x300 != 0 {
					j4 := class26_1.xPos - xCameraPos
					l5 := class26_1.zPos - zCameraPos
					k6 := class26_1.yPos - yCameraPos
					i8 := class26_1.orientation2
					var k9 int
					if i8 == 1 || i8 == 2 {
						k9 = -j4
					} else {
						k9 = j4
					}
					var k10 int
					if i8 == 2 || i8 == 3 {
						k10 = -k6
					} else {
						k10 = k6
					}
					if class26_1.orientation&0x100 != 0 && k10 < k9 {
						i11 := j4 + anIntArray463[i8]
						k11 := k6 + anIntArray464[i8]
						class26_1.renderable.renderAtPoint(i8*512+256, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, i11, l5, k11, class26_1.uid)
					}
					if class26_1.orientation&0x200 != 0 && k10 > k9 {
						j11 := j4 + anIntArray465[i8]
						l11 := k6 + anIntArray466[i8]
						class26_1.renderable.renderAtPoint((i8*512+1280)&0x7ff, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, j11, l5, l11, class26_1.uid)
					}
				}
			}
			if renderDecoration {
				class49 := currentTile.groundDecoration
				if class49 != nil {
					class49.renderable.renderAtPoint(0, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class49.xPos-xCameraPos, class49.zPos-zCameraPos, class49.yPos-yCameraPos, class49.uid)
				}
			}
			k4 := currentTile.totalTiledObjectMask
			if k4 != 0 {
				if x < xCameraTile && k4&4 != 0 {
					class30_sub3_17 := tileHeights[x+1][y]
					if class30_sub3_17 != nil && class30_sub3_17.drawn {
						tileDeque.insertHead(class30_sub3_17)
					}
				}
				if y < yCameraTile && k4&2 != 0 {
					class30_sub3_18 := tileHeights[x][y+1]
					if class30_sub3_18 != nil && class30_sub3_18.drawn {
						tileDeque.insertHead(class30_sub3_18)
					}
				}
				if x > xCameraTile && k4&1 != 0 {
					class30_sub3_19 := tileHeights[x-1][y]
					if class30_sub3_19 != nil && class30_sub3_19.drawn {
						tileDeque.insertHead(class30_sub3_19)
					}
				}
				if y > yCameraTile && k4&8 != 0 {
					class30_sub3_20 := tileHeights[x][y-1]
					if class30_sub3_20 != nil && class30_sub3_20.drawn {
						tileDeque.insertHead(class30_sub3_20)
					}
				}
			}
		}
		if currentTile.renderMask != 0 {
			flag2 := true
			for k1 := 0; k1 < currentTile.gameObjectIndex; k1++ {
				if <<unimp_obj.nm_*parser.GoArrayReference>> == renderedObjectCount || currentTile[k1]&currentTile.renderMask != currentTile.anInt1326 {
					continue
				}
				flag2 = false
				break
			}
			if flag2 {
				class10_1 := currentTile.wallObject
				if !rcvr.method321(plane, x, y, class10_1.orientation1) {
					class10_1.renderable1.renderAtPoint(0, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class10_1.xPos-xCameraPos, class10_1.zPos-zCameraPos, class10_1.yPos-yCameraPos, class10_1.uid)
				}
				currentTile.renderMask = 0
			}
		}
		if currentTile.multipleObjects {
			if try() {
				i1 := currentTile.gameObjectIndex
				currentTile.multipleObjects = false
				l1 := 0
			label0:
				for k2 := 0; k2 < i1; k2++ {
					class28_1 := currentTile[k2]
					if class28_1.rendered == renderedObjectCount {
						continue
					}
					for k3 := class28_1.xLocLow; k3 <= class28_1.xLocHigh; k3++ {
						for l4 := class28_1.yLocHigh; l4 <= class28_1.yLocLow; l4++ {
							class30_sub3_21 := tileHeights[k3][l4]
							if class30_sub3_21.updated {
								currentTile.multipleObjects = true
							} else {
								if class30_sub3_21.renderMask == 0 {
									continue
								}
								l6 := 0
								if k3 > class28_1.xLocLow {
									l6++
								}
								if k3 < class28_1.xLocHigh {
									l6 += 4
								}
								if l4 > class28_1.yLocHigh {
									l6 += 8
								}
								if l4 < class28_1.yLocLow {
									l6 += 2
								}
								if l6&class30_sub3_21.renderMask != currentTile.anInt1327 {
									continue
								}
								currentTile.multipleObjects = true
							}
							continue label0
						}
					}
					interactableObjects[++l1] = class28_1
					i5 := xCameraTile - class28_1.xLocLow
					i6 := class28_1.xLocHigh - xCameraTile
					if i6 > i5 {
						i5 = i6
					}
					i7 := yCameraTile - class28_1.yLocHigh
					j8 := class28_1.yLocLow - yCameraTile
					if j8 > i7 {
						class28_1.cameraDistance = i5 + j8
					} else {
						class28_1.cameraDistance = i5 + i7
					}
				}
				for l1 > 0 {
					i3 := -50
					l3 := -1
					for j5 := 0; j5 < l1; j5++ {
						class28_2 := interactableObjects[j5]
						if class28_2.rendered != renderedObjectCount {
							if class28_2.cameraDistance > i3 {
								i3 = class28_2.cameraDistance
								l3 = j5
							} else if class28_2.cameraDistance == i3 {
								j7 := class28_2.x - xCameraPos
								k8 := class28_2.y - yCameraPos
								l9 := <<unimp_obj.nm_*parser.GoArrayReference>> - xCameraPos
								l10 := <<unimp_obj.nm_*parser.GoArrayReference>> - yCameraPos
								if j7*j7+k8*k8 > l9*l9+l10*l10 {
									l3 = j5
								}
							}
						}
					}
					if l3 == -1 {
						break
					}
					class28_3 := interactableObjects[l3]
					class28_3.rendered = renderedObjectCount
					if !rcvr.method323(plane, class28_3.xLocLow, class28_3.xLocHigh, class28_3.yLocHigh, class28_3.yLocLow, class28_3.renderable.modelBaseY) {
						class28_3.renderable.renderAtPoint(class28_3.turnValue, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class28_3.x-xCameraPos, class28_3.tileHeight-zCameraPos, class28_3.y-yCameraPos, class28_3.uid)
					}
					for k7 := class28_3.xLocLow; k7 <= class28_3.xLocHigh; k7++ {
						for l8 := class28_3.yLocHigh; l8 <= class28_3.yLocLow; l8++ {
							class30_sub3_22 := tileHeights[k7][l8]
							if class30_sub3_22.renderMask != 0 {
								tileDeque.insertHead(class30_sub3_22)
							} else if (k7 != x || l8 != y) && class30_sub3_22.drawn {
								tileDeque.insertHead(class30_sub3_22)
							}
						}
					}
				}
				if currentTile.multipleObjects {
					continue
				}
			} else if catch_Exception(_ex) {
				currentTile.multipleObjects = false
			}
		}
		if !currentTile.drawn || currentTile.renderMask != 0 {
			continue
		}
		if x <= xCameraTile && x > cameraLowTileX {
			class30_sub3_8 := tileHeights[x-1][y]
			if class30_sub3_8 != nil && class30_sub3_8.drawn {
				continue
			}
		}
		if x >= xCameraTile && x < cameraHighTileX-1 {
			class30_sub3_9 := tileHeights[x+1][y]
			if class30_sub3_9 != nil && class30_sub3_9.drawn {
				continue
			}
		}
		if y <= yCameraTile && y > cameraLowTileY {
			class30_sub3_10 := tileHeights[x][y-1]
			if class30_sub3_10 != nil && class30_sub3_10.drawn {
				continue
			}
		}
		if y >= yCameraTile && y < cameraHighTileY-1 {
			class30_sub3_11 := tileHeights[x][y+1]
			if class30_sub3_11 != nil && class30_sub3_11.drawn {
				continue
			}
		}
		currentTile.drawn = false
		clickCount--
		if currentTile.anInt1328 != 0 {
			class26 := currentTile.wallDecoration
			if class26 != nil && !rcvr.method322(plane, x, y, class26.renderable.modelBaseY) {
				if class26.orientation&currentTile.anInt1328 != 0 {
					class26.renderable.renderAtPoint(class26.orientation2, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class26.xPos-xCameraPos, class26.zPos-zCameraPos, class26.yPos-yCameraPos, class26.uid)
				} else if class26.orientation&0x300 != 0 {
					l2 := class26.xPos - xCameraPos
					j3 := class26.zPos - zCameraPos
					i4 := class26.yPos - yCameraPos
					k5 := class26.orientation2
					var j6 int
					if k5 == 1 || k5 == 2 {
						j6 = -l2
					} else {
						j6 = l2
					}
					var l7 int
					if k5 == 2 || k5 == 3 {
						l7 = -i4
					} else {
						l7 = i4
					}
					if class26.orientation&0x100 != 0 && l7 >= j6 {
						i9 := l2 + anIntArray463[k5]
						i10 := i4 + anIntArray464[k5]
						class26.renderable.renderAtPoint(k5*512+256, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, i9, j3, i10, class26.uid)
					}
					if class26.orientation&0x200 != 0 && l7 <= j6 {
						j9 := l2 + anIntArray465[k5]
						j10 := i4 + anIntArray466[k5]
						class26.renderable.renderAtPoint((k5*512+1280)&0x7ff, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, j9, j3, j10, class26.uid)
					}
				}
			}
			class10_2 := currentTile.wallObject
			if class10_2 != nil {
				if class10_2.orientation2&currentTile.anInt1328 != 0 && !rcvr.method321(plane, x, y, class10_2.orientation2) {
					class10_2.renderable2.renderAtPoint(0, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class10_2.xPos-xCameraPos, class10_2.zPos-zCameraPos, class10_2.yPos-yCameraPos, class10_2.uid)
				}
				if class10_2.orientation1&currentTile.anInt1328 != 0 && !rcvr.method321(plane, x, y, class10_2.orientation1) {
					class10_2.renderable1.renderAtPoint(0, camUpDownY, camUpDownX, camLeftRightY, camLeftRightX, class10_2.xPos-xCameraPos, class10_2.zPos-zCameraPos, class10_2.yPos-yCameraPos, class10_2.uid)
				}
			}
		}
		if z < rcvr.zRegionSize-1 {
			class30_sub3_12 := tileArray[z+1][x][y]
			if class30_sub3_12 != nil && class30_sub3_12.drawn {
				tileDeque.insertHead(class30_sub3_12)
			}
		}
		if x < xCameraTile {
			class30_sub3_13 := tileHeights[x+1][y]
			if class30_sub3_13 != nil && class30_sub3_13.drawn {
				tileDeque.insertHead(class30_sub3_13)
			}
		}
		if y < yCameraTile {
			class30_sub3_14 := tileHeights[x][y+1]
			if class30_sub3_14 != nil && class30_sub3_14.drawn {
				tileDeque.insertHead(class30_sub3_14)
			}
		}
		if x > xCameraTile {
			class30_sub3_15 := tileHeights[x-1][y]
			if class30_sub3_15 != nil && class30_sub3_15.drawn {
				tileDeque.insertHead(class30_sub3_15)
			}
		}
		if y > yCameraTile {
			class30_sub3_16 := tileHeights[x][y-1]
			if class30_sub3_16 != nil && class30_sub3_16.drawn {
				tileDeque.insertHead(class30_sub3_16)
			}
		}
		if !(true) {
			break
		}
	}
}
func (rcvr *SceneGraph) SetTileLogicHeight(zLoc int, xLoc int, yLoc int, logicHeight int) {
	tile := tileArray[zLoc][xLoc][yLoc]
	if tile != nil {
		<<unimp_obj.nm_*parser.GoArrayReference>> = logicHeight
	}
}
func SetupViewport(minimumZ int, maximumZ int, viewportWidth int, viewportHeight int, ai []int) {
	anInt495 = 0
	anInt496 = 0
	SceneGraph.viewportWidth = viewportWidth
	SceneGraph.viewportHeight = viewportHeight
	viewportHalfWidth = viewportWidth / 2
	viewportHalfHeight = viewportHeight / 2
	aflag := make([]bool, 9, 32, 53, 53)
	for zAngle := 128; zAngle <= 384; zAngle += 32 {
		for xyAngle := 0; xyAngle < 2048; xyAngle += 64 {
			camUpDownY = <<unimp_arrayref_Model.sine>>[zAngle]
			camUpDownX = <<unimp_arrayref_Model.cosine>>[zAngle]
			camLeftRightY = <<unimp_arrayref_Model.sine>>[xyAngle]
			camLeftRightX = <<unimp_arrayref_Model.cosine>>[xyAngle]
			angularZSegment := (zAngle - 128) / 32
			angularXYSegment := xyAngle / 64
			for xRelativeToCamera := -26; xRelativeToCamera <= 26; xRelativeToCamera++ {
				for yRelativeToCamera := -26; yRelativeToCamera <= 26; yRelativeToCamera++ {
					xRelativeToCameraPos := xRelativeToCamera * 128
					yRelativeToCameraPos := yRelativeToCamera * 128
					flag2 := false
					for zRelativeCameraPos := -minimumZ; zRelativeCameraPos <= maximumZ; zRelativeCameraPos += 128 {
						if !SceneGraph.method311(ai[angularZSegment]+zRelativeCameraPos, yRelativeToCameraPos, xRelativeToCameraPos) {
							continue
						}
						flag2 = true
						break
					}
					aflag[angularZSegment][angularXYSegment][xRelativeToCamera+25+1][yRelativeToCamera+25+1] = flag2
				}
			}
		}
	}
	for angularZSegment := 0; angularZSegment < 8; angularZSegment++ {
		for angularXYSegment := 0; angularXYSegment < 32; angularXYSegment++ {
			for xRelativeToCamera := -25; xRelativeToCamera < 25; xRelativeToCamera++ {
				for yRelativeToCamera := -25; yRelativeToCamera < 25; yRelativeToCamera++ {
					flag1 := false
				label0:
					for l3 := -1; l3 <= 1; l3++ {
						for j4 := -1; j4 <= 1; j4++ {
							if aflag[angularZSegment][angularXYSegment][xRelativeToCamera+l3+25+1][yRelativeToCamera+j4+25+1] {
								flag1 = true
							} else if aflag[angularZSegment][(angularXYSegment+1)%31][xRelativeToCamera+l3+25+1][yRelativeToCamera+j4+25+1] {
								flag1 = true
							} else if aflag[angularZSegment+1][angularXYSegment][xRelativeToCamera+l3+25+1][yRelativeToCamera+j4+25+1] {
								flag1 = true
							} else {
								if !aflag[angularZSegment+1][(angularXYSegment+1)%31][xRelativeToCamera+l3+25+1][yRelativeToCamera+j4+25+1] {
									continue
								}
								flag1 = true
							}
							break label0
						}
					}
					aBooleanArrayArrayArrayArray491[angularZSegment][angularXYSegment][xRelativeToCamera+25][yRelativeToCamera+25] = flag1
				}
			}
		}
	}
}
func (rcvr *SceneGraph) ShadeModels(lightY int, lightX int, lightZ int) {
	intensity := 85
	diffusion := 768
	lightDistance, ok := Math.sqrt(lightX*lightX + lightY*lightY + lightZ*lightZ).(int)
	if !ok {
		panic("XXX Cast fail for *parser.GoCastType")
	}
	someLightQualityVariable := uint32(diffusion*lightDistance) >> 8
	for zLoc := 0; zLoc < rcvr.zRegionSize; zLoc++ {
		for xLoc := 0; xLoc < rcvr.xRegionSize; xLoc++ {
			for yLoc := 0; yLoc < rcvr.yRegionSize; yLoc++ {
				tile := tileArray[zLoc][xLoc][yLoc]
				if tile != nil {
					wallObject := tile.wallObject
					if wallObject != nil && wallObject.renderable1 != nil && wallObject.renderable1.vertexNormals != nil {
						rcvr.method307(zLoc, 1, 1, xLoc, yLoc, wallObject.renderable1.(*Model))
						if wallObject.renderable2 != nil && wallObject.renderable2.vertexNormals != nil {
							rcvr.method307(zLoc, 1, 1, xLoc, yLoc, wallObject.renderable2.(*Model))
							rcvr.mergeNormals(wallObject.renderable1.(*Model), wallObject.renderable2.(*Model), 0, 0, 0, false)
							wallObject.renderable2.(*Model).flatLighting(intensity, someLightQualityVariable, lightX, lightY, lightZ)
						}
						wallObject.renderable1.(*Model).flatLighting(intensity, someLightQualityVariable, lightX, lightY, lightZ)
					}
					for k2 := 0; k2 < tile.gameObjectIndex; k2++ {
						interactableObject := tile[k2]
						if interactableObject != nil && interactableObject.renderable != nil && interactableObject.renderable.vertexNormals != nil {
							method307(zLoc, interactableObject.xLocHigh-interactableObject.xLocLow+1, interactableObject.yLocLow-interactableObject.yLocHigh+1, xLoc, yLoc, interactableObject.renderable.(*Model))
							interactableObject.renderable.(*Model).flatLighting(intensity, someLightQualityVariable, lightX, lightY, lightZ)
						}
					}
					groundDecoration := tile.groundDecoration
					if groundDecoration != nil && groundDecoration.renderable.vertexNormals != nil {
						rcvr.method306GroundDecorationOnly(xLoc, zLoc, groundDecoration.renderable.(*Model), yLoc)
						groundDecoration.renderable.(*Model).flatLighting(intensity, someLightQualityVariable, lightX, lightY, lightZ)
					}
				}
			}
		}
	}
}
func (rcvr *SceneGraph) tileVisible(zLoc int, xLoc int, yLoc int) (bool) {
	currentRenderedViewableObjects := renderedViewableObjects[zLoc][xLoc][yLoc]
	if currentRenderedViewableObjects == -renderedObjectCount {
		return false
	}
	if currentRenderedViewableObjects == renderedObjectCount {
		return true
	}
	xPos := xLoc << 7
	yPos := yLoc << 7
	if rcvr.visible(xPos+1, heightMap[zLoc][xLoc][yLoc], yPos+1) && visible(xPos+128-1, heightMap[zLoc][xLoc+1][yLoc], yPos+1) && visible(xPos+128-1, heightMap[zLoc][xLoc+1][yLoc+1], yPos+128-1) && visible(xPos+1, heightMap[zLoc][xLoc][yLoc+1], yPos+128-1) {
		renderedViewableObjects[zLoc][xLoc][yLoc] = renderedObjectCount
		return true
	} else {
		renderedViewableObjects[zLoc][xLoc][yLoc] = -renderedObjectCount
		return false
	}
}
func (rcvr *SceneGraph) visible(i int, j int, k int) (bool) {
	for l := 0; l < processedCullingCluster; l++ {
		class47 := fixedCullingClusters[l]
		if class47.cullDirection == 1 {
			i1 := class47.startXPos - i
			if i1 > 0 {
				j2 := class47.startYPos + uint32(class47.cameraDistanceStartY*i1)>>8
				k3 := class47.endYPos + uint32(class47.cameraDistanceEndY*i1)>>8
				l4 := class47.startZPos + uint32(class47.cameraDistanceStartZ*i1)>>8
				i6 := class47.endZPos + uint32(class47.cameraDistanceEndZ*i1)>>8
				if k >= j2 && k <= k3 && j >= l4 && j <= i6 {
					return true
				}
			}
		} else if class47.cullDirection == 2 {
			j1 := i - class47.startXPos
			if j1 > 0 {
				k2 := class47.startYPos + uint32(class47.cameraDistanceStartY*j1)>>8
				l3 := class47.endYPos + uint32(class47.cameraDistanceEndY*j1)>>8
				i5 := class47.startZPos + uint32(class47.cameraDistanceStartZ*j1)>>8
				j6 := class47.endZPos + uint32(class47.cameraDistanceEndZ*j1)>>8
				if k >= k2 && k <= l3 && j >= i5 && j <= j6 {
					return true
				}
			}
		} else if class47.cullDirection == 3 {
			k1 := class47.startYPos - k
			if k1 > 0 {
				l2 := class47.startXPos + uint32(class47.cameraDistanceStartX*k1)>>8
				i4 := class47.endXPos + uint32(class47.cameraDistanceEndX*k1)>>8
				j5 := class47.startZPos + uint32(class47.cameraDistanceStartZ*k1)>>8
				k6 := class47.endZPos + uint32(class47.cameraDistanceEndZ*k1)>>8
				if i >= l2 && i <= i4 && j >= j5 && j <= k6 {
					return true
				}
			}
		} else if class47.cullDirection == 4 {
			l1 := k - class47.startYPos
			if l1 > 0 {
				i3 := class47.startXPos + uint32(class47.cameraDistanceStartX*l1)>>8
				j4 := class47.endXPos + uint32(class47.cameraDistanceEndX*l1)>>8
				k5 := class47.startZPos + uint32(class47.cameraDistanceStartZ*l1)>>8
				l6 := class47.endZPos + uint32(class47.cameraDistanceEndZ*l1)>>8
				if i >= i3 && i <= j4 && j >= k5 && j <= l6 {
					return true
				}
			}
		} else if class47.cullDirection == 5 {
			i2 := j - class47.startZPos
			if i2 > 0 {
				j3 := class47.startXPos + uint32(class47.cameraDistanceStartX*i2)>>8
				k4 := class47.endXPos + uint32(class47.cameraDistanceEndX*i2)>>8
				l5 := class47.startYPos + uint32(class47.cameraDistanceStartY*i2)>>8
				i7 := class47.endYPos + uint32(class47.cameraDistanceEndY*i2)>>8
				if i >= j3 && i <= k4 && k >= l5 && k <= i7 {
					return true
				}
			}
		}
	}
	return false
}

var AnIntArray688 = make([]int, 6)
var AnIntArray689 = make([]int, 6)
var AnIntArray690 = make([]int, 6)
var AnIntArray691 = make([]int, 6)
var AnIntArray692 = make([]int, 6)
var DepthPoint = make([]int, 6)
var anIntArrayArray696 []int
var anIntArrayArray697 []int

type ShapedTile struct {
	AnIntArray673 []int
	AnIntArray674 []int
	AnIntArray675 []int
	AnIntArray682 []int
	AnIntArray679 []int
	AnIntArray680 []int
	AnIntArray681 []int
	AnIntArray677 []int
	AnIntArray678 []int
	AnIntArray676 []int
	Flat          bool
}

func NewShapedTile(yLoc int, j int, k int, l int, texture int, j1 int, rotation int, l1 int, i2 int, j2 int, k2 int, l2 int, i3 int, j3 int, k3 int, l3 int, i4 int, xLoc int, l4 int) (rcvr *ShapedTile) {
	rcvr = &ShapedTile{}
	rcvr.Flat = !(i3 != l2 || i3 != l || i3 != k2)
	sideLength := 128
	halfSizeLength := sideLength / 2
	quarterSizeLight := sideLength / 4
	k5 := sideLength * 3 / 4
	ai := anIntArrayArray696[j3]
	l5 := len(ai)
	rcvr.AnIntArray673 = make([]int, l5)
	rcvr.AnIntArray674 = make([]int, l5)
	rcvr.AnIntArray675 = make([]int, l5)
	ai1 := make([]int, l5)
	ai2 := make([]int, l5)
	xPos := xLoc * sideLength
	yPos := yLoc * sideLength
	for k6 := 0; k6 < l5; k6++ {
		realShape := ai[k6]
		if realShape&1 == 0 && realShape <= 8 {
			realShape = (realShape-rotation-rotation-1)&7 + 1
		}
		if realShape > 8 && realShape <= 12 {
			realShape = (realShape-9-rotation)&3 + 9
		}
		if realShape > 12 && realShape <= 16 {
			realShape = (realShape-13-rotation)&3 + 13
		}
		var i7 int
		var k7 int
		var i8 int
		var k8 int
		var j9 int
		if realShape == 1 {
			i7 = xPos
			k7 = yPos
			i8 = i3
			k8 = l1
			j9 = j
		} else if realShape == 2 {
			i7 = xPos + halfSizeLength
			k7 = yPos
			i8 = uint32(i3+l2) >> 1
			k8 = uint32(l1+i4) >> 1
			j9 = uint32(j+l3) >> 1
		} else if realShape == 3 {
			i7 = xPos + sideLength
			k7 = yPos
			i8 = l2
			k8 = i4
			j9 = l3
		} else if realShape == 4 {
			i7 = xPos + sideLength
			k7 = yPos + halfSizeLength
			i8 = uint32(l2+l) >> 1
			k8 = uint32(i4+j2) >> 1
			j9 = uint32(l3+j1) >> 1
		} else if realShape == 5 {
			i7 = xPos + sideLength
			k7 = yPos + sideLength
			i8 = l
			k8 = j2
			j9 = j1
		} else if realShape == 6 {
			i7 = xPos + halfSizeLength
			k7 = yPos + sideLength
			i8 = uint32(l+k2) >> 1
			k8 = uint32(j2+k) >> 1
			j9 = uint32(j1+k3) >> 1
		} else if realShape == 7 {
			i7 = xPos
			k7 = yPos + sideLength
			i8 = k2
			k8 = k
			j9 = k3
		} else if realShape == 8 {
			i7 = xPos
			k7 = yPos + halfSizeLength
			i8 = uint32(k2+i3) >> 1
			k8 = uint32(k+l1) >> 1
			j9 = uint32(k3+j) >> 1
		} else if realShape == 9 {
			i7 = xPos + halfSizeLength
			k7 = yPos + quarterSizeLight
			i8 = uint32(i3+l2) >> 1
			k8 = uint32(l1+i4) >> 1
			j9 = uint32(j+l3) >> 1
		} else if realShape == 10 {
			i7 = xPos + k5
			k7 = yPos + halfSizeLength
			i8 = uint32(l2+l) >> 1
			k8 = uint32(i4+j2) >> 1
			j9 = uint32(l3+j1) >> 1
		} else if realShape == 11 {
			i7 = xPos + halfSizeLength
			k7 = yPos + k5
			i8 = uint32(l+k2) >> 1
			k8 = uint32(j2+k) >> 1
			j9 = uint32(j1+k3) >> 1
		} else if realShape == 12 {
			i7 = xPos + quarterSizeLight
			k7 = yPos + halfSizeLength
			i8 = uint32(k2+i3) >> 1
			k8 = uint32(k+l1) >> 1
			j9 = uint32(k3+j) >> 1
		} else if realShape == 13 {
			i7 = xPos + quarterSizeLight
			k7 = yPos + quarterSizeLight
			i8 = i3
			k8 = l1
			j9 = j
		} else if realShape == 14 {
			i7 = xPos + k5
			k7 = yPos + quarterSizeLight
			i8 = l2
			k8 = i4
			j9 = l3
		} else if realShape == 15 {
			i7 = xPos + k5
			k7 = yPos + k5
			i8 = l
			k8 = j2
			j9 = j1
		} else {
			i7 = xPos + quarterSizeLight
			k7 = yPos + k5
			i8 = k2
			k8 = k
			j9 = k3
		}
		AnIntArray673[k6] = i7
		AnIntArray674[k6] = i8
		AnIntArray675[k6] = k7
		ai1[k6] = k8
		ai2[k6] = j9
	}
	ai3 := anIntArrayArray697[j3]
	j7 := len(ai3) / 4
	rcvr.AnIntArray679 = make([]int, j7)
	rcvr.AnIntArray680 = make([]int, j7)
	rcvr.AnIntArray681 = make([]int, j7)
	rcvr.AnIntArray676 = make([]int, j7)
	rcvr.AnIntArray677 = make([]int, j7)
	rcvr.AnIntArray678 = make([]int, j7)
	if texture != -1 {
		rcvr.AnIntArray682 = make([]int, j7)
	}
	l7 := 0
	for j8 := 0; j8 < j7; j8++ {
		l8 := ai3[l7]
		k9 := ai3[l7+1]
		i10 := ai3[l7+2]
		k10 := ai3[l7+3]
		l7 += 4
		if k9 < 4 {
			k9 = (k9 - rotation) & 3
		}
		if i10 < 4 {
			i10 = (i10 - rotation) & 3
		}
		if k10 < 4 {
			k10 = (k10 - rotation) & 3
		}
		AnIntArray679[j8] = k9
		AnIntArray680[j8] = i10
		AnIntArray681[j8] = k10
		if l8 == 0 {
			AnIntArray676[j8] = ai1[k9]
			AnIntArray677[j8] = ai1[i10]
			AnIntArray678[j8] = ai1[k10]
			if rcvr.AnIntArray682 != nil {
				AnIntArray682[j8] = -1
			}
		} else {
			AnIntArray676[j8] = ai2[k9]
			AnIntArray677[j8] = ai2[i10]
			AnIntArray678[j8] = ai2[k10]
			if rcvr.AnIntArray682 != nil {
				AnIntArray682[j8] = texture
			}
		}
	}
	i9 := i3
	l9 := l2
	if l2 < i9 {
		i9 = l2
	}
	if l2 > l9 {
		l9 = l2
	}
	if l < i9 {
		i9 = l
	}
	if l > l9 {
		l9 = l
	}
	if k2 < i9 {
		i9 = k2
	}
	if k2 > l9 {
		l9 = k2
	}
	i9 /= 14
	l9 /= 14
	return
}

type SimpleTile struct {
	northEastColor int
	northColor     int
	centerColor    int
	eastColor      int
	texture        int
	flat           bool
	colorRGB       int
}

func NewSimpleTile(northEastColor int, northColor int, centerColor int, eastColor int, texture int, colorRGB int, flat bool) (rcvr *SimpleTile) {
	rcvr = &SimpleTile{}
	rcvr.northEastColor = northEastColor
	rcvr.northColor = northColor
	rcvr.centerColor = centerColor
	rcvr.eastColor = eastColor
	rcvr.texture = texture
	rcvr.colorRGB = colorRGB
	rcvr.flat = flat
	return
}
func (rcvr *SimpleTile) GetCenterColor() (int) {
	return rcvr.centerColor
}
func (rcvr *SimpleTile) GetColourRGB() (int) {
	return rcvr.colorRGB
}
func (rcvr *SimpleTile) GetEastColor() (int) {
	return rcvr.eastColor
}
func (rcvr *SimpleTile) GetNorthColor() (int) {
	return rcvr.northColor
}
func (rcvr *SimpleTile) GetNorthEastColor() (int) {
	return rcvr.northEastColor
}
func (rcvr *SimpleTile) GetTexture() (int) {
	return rcvr.texture
}
func (rcvr *SimpleTile) IsFlat() (bool) {
	return rcvr.flat
}

type Tile struct {
	*Linkable
	LogicHeight          int
	Updated              bool
	Drawn                bool
	RenderMask           int
	MultipleObjects      bool
	GameObjectIndex      int
	X                    int
	Y                    int
	Z                    int
	Plane                int
	TotalTiledObjectMask int
	FirstFloorTile       *Tile
	MySimpleTile         *SimpleTile
	MyShapedTile         *ShapedTile
	WallObject           *WallObject
	GameObjects          []*GameObject
	WallDecoration       *WallDecoration
	AnInt1326            int
	AnInt1327            int
	AnInt1328            int
	GroundDecoration     *GroundDecoration
	TiledObjectMasks     []int
}

func NewTile(zLoc int, xLoc int, yLoc int) (rcvr *Tile) {
	rcvr = &Tile{}
	rcvr.GameObjects = make([]*GameObject, 5)
	rcvr.TiledObjectMasks = make([]int, 5)
	rcvr.Plane = rcvr.Z = zLoc
	rcvr.X = xLoc
	rcvr.Y = yLoc
	return
}

type VertexNormal struct {
	NormalX   int
	NormalY   int
	NormalZ   int
	Magnitude int
}

func NewVertexNormal() (rcvr *VertexNormal) {
	rcvr = &VertexNormal{}
	return
}

type WallDecoration struct {
	Orientation  int
	Renderable   *Renderable
	Orientation2 int
	Uid          int
	XPos         int
	ZPos         int
	YPos         int
	Mask         byte
}

func NewWallDecoration() (rcvr *WallDecoration) {
	rcvr = &WallDecoration{}
	return
}

type WallObject struct {
	Renderable1  *Renderable
	XPos         int
	ZPos         int
	YPos         int
	Uid          int
	Orientation1 int
	Orientation2 int
	Renderable2  *Renderable
	Mask         byte
}

func NewWallObject() (rcvr *WallObject) {
	rcvr = &WallObject{}
	return
}
