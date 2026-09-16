package mime

import (
	"mime"
	"path/filepath"
	"sort"
	"strings"
)

// extensionMap contains project-specific extension overrides for common formats.
// Extensions not listed here are resolved through the host MIME database.
var extensionMap = map[string]Type{
	// Application
	".7z":           ApplicationOctetStream,
	".app":          ApplicationOctetStream,
	".atom":         ApplicationAtomXML,
	".bin":          ApplicationOctetStream,
	".cer":          ApplicationX509CACert,
	".css":          TextCSS,
	".crt":          ApplicationX509CACert,
	".der":          ApplicationX509CACert,
	".dmg":          ApplicationOctetStream,
	".doc":          ApplicationMSWord,
	".docx":         ApplicationMSWord,
	".epub":         ApplicationEPUB,
	".exe":          ApplicationOctetStream,
	".gguf":         ApplicationOctetStream,
	".gpkg":         ApplicationGeoPackage,
	".gz":           ApplicationGzip,
	".js":           TextJavaScript,
	".jar":          ApplicationJavaArchive,
	".key":          ApplicationOctetStream,
	".json":         ApplicationJSON,
	".jsonl":        ApplicationJSONSeq,
	".map":          ApplicationJSON,
	".mjs":          TextJavaScript,
	".msi":          ApplicationOctetStream,
	".ndjson":       ApplicationJSONSeq,
	".p12":          ApplicationPKCS12,
	".p7b":          ApplicationPKIXCert,
	".p7c":          ApplicationPKIXCert,
	".parquet":      ApplicationParquet,
	".pem":          ApplicationPEMCertificateChain,
	".pdf":          ApplicationPDF,
	".pfx":          ApplicationPKCS12,
	".ipa":          ApplicationZip,
	".icns":         ApplicationOctetStream,
	".mobileconfig": ApplicationXML,
	".numbers":      ApplicationOctetStream,
	".pages":        ApplicationOctetStream,
	".pkg":          ApplicationOctetStream,
	".plist":        ApplicationXML,
	".ppt":          ApplicationMSPowerPoint,
	".pptx":         ApplicationMSPowerPoint,
	".ps":           ApplicationPostScript,
	".proto":        ApplicationProtobuf,
	".rdf":          ApplicationRDF,
	".rss":          ApplicationAtomXML,
	".safetensors":  ApplicationOctetStream,
	".sqlite":       ApplicationSQLite3,
	".sqlite3":      ApplicationSQLite3,
	".toml":         ApplicationTOML,
	".tar":          ApplicationOctetStream,
	".wasm":         ApplicationWASM,
	".webmanifest":  ApplicationManifest,
	".xls":          ApplicationMSExcel,
	".xlsx":         ApplicationMSExcel,
	".xml":          ApplicationXML,
	".yaml":         ApplicationYAML,
	".yml":          ApplicationYAML,
	".zip":          ApplicationZip,
	".zst":          ApplicationZStd,

	// Audio
	".3ga":  Audio3GPP,
	".aac":  AudioAAC,
	".ac3":  AudioAC3,
	".amr":  AudioAMR,
	".au":   AudioBasic,
	".flac": AudioFLAC,
	".m4a":  AudioMP4,
	".m4b":  AudioMP4,
	".m4p":  AudioMP4,
	".mid":  AudioMIDIClip,
	".midi": AudioMIDIClip,
	".mp2":  AudioMPEG,
	".mp3":  AudioMPEG,
	".oga":  AudioOGG,
	".ogg":  AudioOGG,
	".opus": AudioOpus,
	".sf2":  AudioSoundFont,
	".spx":  AudioSpeex,

	// Fonts
	".otf":   FontOTF,
	".ttc":   FontCollection,
	".ttf":   FontTTF,
	".woff":  FontWOFF,
	".woff2": FontWOFF2,

	// Images
	".apng":  ImageAPNG,
	".avif":  ImageAVIF,
	".bmp":   ImageBMP,
	".cgm":   ImageCGM,
	".gif":   ImageGIF,
	".heic":  ImageHEIC,
	".heics": ImageHEICSequence,
	".heif":  ImageHEIF,
	".heifs": ImageHEIFSequence,
	".ico":   ImageIcon,
	".jpeg":  ImageJPEG,
	".jpg":   ImageJPEG,
	".png":   ImagePNG,
	".psd":   ImagePhotoshop,
	".svg":   ImageSVG,
	".tif":   ImageTIFF,
	".tiff":  ImageTIFF,
	".webp":  ImageWebP,
	".wmf":   ImageWMF,

	// Models
	".3mf":  Model3MF,
	".e57":  ModelE57,
	".glb":  ModelGLTF,
	".gltf": ModelGLTFJSON,
	".iges": ModelIGES,
	".igs":  ModelIGES,
	".jt":   ModelJT,
	".mesh": ModelMesh,
	".mtl":  ModelMTL,
	".obj":  ModelObj,
	".prc":  ModelPRC,
	".step": ModelStep,
	".stl":  ModelSTL,
	".stp":  ModelStep,
	".u3d":  ModelU3D,
	".vrml": ModelVRML,
	".wrl":  ModelVRML,
	".x3d":  ModelX3D,

	// Text
	".cql":   TextCQL,
	".csv":   TextCSV,
	".htm":   TextHTML,
	".html":  TextHTML,
	".latex": TextLatexZ,
	".md":    TextMarkdown,
	".org":   TextOrg,
	".rtf":   TextRTF,
	".tsv":   TextTabSeparatedValues,
	".txt":   TextPlain,
	".ttl":   TextTurtle,
	".vtt":   TextVTT,
	".wgsl":  TextWGSL,
	".xhtml": TextHTML,

	// Message
	".eml": MessageRFC822,

	// Multipart
	".mhtml": MultipartRelated,

	// Video
	".3g2":  Video3GPP2,
	".3gp":  Video3GPP,
	".h261": VideoH261,
	".h263": VideoH263,
	".h264": VideoH264,
	".h265": VideoH265,
	".h266": VideoH266,
	".m2v":  VideoMPEG,
	".m4v":  VideoMP4,
	".mov":  VideoQuickTime,
	".mp4":  VideoMP4,
	".mpeg": VideoMPEG,
	".mpg":  VideoMPEG,
	".ogv":  VideoOGG,
	".vp8":  VideoVP8,
	".vp9":  VideoVP9,
}

// TypeByExtension returns the media type for a file extension.
func TypeByExtension(path string) Type {
	// Lookup Endeavor known file extensions.
	ext := strings.ToLower(filepath.Ext(path))
	if t, ok := extensionMap[ext]; ok {
		return t
	}

	// Use host operating system's MIME type database to determine the media type.
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return Type(contentType)
	}

	// Return unknown media type.
	return UnknownMimeType
}

// ExtensionsByType returns the known file extensions for a media type.
func ExtensionsByType(mediaType Type) []string {
	extensions := make([]string, 0)
	for extension, typ := range extensionMap {
		if typ == mediaType {
			extensions = append(extensions, extension)
		}
	}
	sort.Strings(extensions)
	return extensions
}
