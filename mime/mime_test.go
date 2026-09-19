package mime_test

import (
	"strings"
	"testing"

	"go.rtnl.ai/x/assert"
	. "go.rtnl.ai/x/mime"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected MimeType
	}{
		{
			name:  "Simple",
			input: "text/plain",
			expected: MimeType{
				Type:   TextPlain,
				Params: map[string]string{},
			},
		},
		{
			name:  "WithParams",
			input: "text/plain; charset=utf-8",
			expected: MimeType{
				Type:   TextPlain,
				Params: map[string]string{"charset": "utf-8"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := Parse(tc.input)
			assert.Ok(t, err)
			assert.Equal(t, tc.expected.Type, actual.Type)
			assert.Equal(t, tc.expected.Params, actual.Params)
		})
	}

	t.Run("Invalid", func(t *testing.T) {
		_, err := Parse("text/plain; charset=utf-8; invalid")
		assert.Error(t, err)
	})

	t.Run("NoSubtype", func(t *testing.T) {
		_, err := Parse("text/")
		assert.Error(t, err)
	})
}

func TestDetectType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected Type
	}{
		{
			name:     "Binary",
			data:     []byte{0x00, 0x01, 0x02},
			expected: ApplicationOctetStream,
		},
		{
			name:     "Text",
			data:     []byte("hello world\n"),
			expected: Type("text/plain; charset=utf-8"),
		},
		{
			name:     "PNG",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: ImagePNG,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, DetectType(tc.data))
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		input    Type
		expected error
	}{
		{
			name:     "Valid",
			input:    TextPlain,
			expected: nil,
		},
		{
			name:     "Empty",
			input:    "",
			expected: ErrEmptyMimeType,
		},
		{
			name:     "Invalid",
			input:    "text/",
			expected: ErrInvalidMimeType,
		},
		{
			name:     "Params",
			input:    "text/plain; charset=utf-8",
			expected: ErrInvalidMimeType,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.expected == nil {
				assert.Ok(t, err)
			} else {
				assert.ErrorIs(t, err, tc.expected)
			}
		})
	}
}

var (
	applicationTypes = []Type{
		ApplicationActiveMessage,
		ApplicationActivityJSON,
		ApplicationAppleFile,
		ApplicationAtomXML,
		ApplicationAuthPolicyXML,
		ApplicationCalendarJSON,
		ApplicationCalendarXML,
		ApplicationDNS,
		ApplicationDNSJSON,
		ApplicationDNSMessage,
		ApplicationEmotionML,
		ApplicationEPUB,
		ApplicationExpress,
		ApplicationGeoJSON,
		ApplicationGeoJSONSeq,
		ApplicationGeoFeed,
		ApplicationGeoPackage,
		ApplicationGzip,
		ApplicationH224,
		ApplicationHTTP,
		ApplicationJavaArchive,
		ApplicationJSON,
		ApplicationJSONSeq,
		ApplicationJSONPath,
		ApplicationJWKJSON,
		ApplicationJWKSetJSON,
		ApplicationJWKSetJWT,
		ApplicationManifest,
		ApplicationMathematica,
		ApplicationMathML,
		ApplicationMBox,
		ApplicationMP21,
		ApplicationMP4,
		ApplicationMPEG4Generic,
		ApplicationMPEG4IOD,
		ApplicationMPEG4IODXMT,
		ApplicationMSWord,
		ApplicationMultipartCore,
		ApplicationNode,
		ApplicationOauthAuthZReq,
		ApplicationOCSPRequest,
		ApplicationOCSPResponse,
		ApplicationOctetStream,
		ApplicationPassport,
		ApplicationPDF,
		ApplicationPDX,
		ApplicationPEMCertificateChain,
		ApplicationPGPEncrypted,
		ApplicationPGPKeys,
		ApplicationPGPSignature,
		ApplicationPKCS8,
		ApplicationPKCS8Encrypted,
		ApplicationPKCS10,
		ApplicationPKCS12,
		ApplicationPKIXAttrCert,
		ApplicationPKIXCert,
		ApplicationPKIXCRL,
		ApplicationPKIXPKIPath,
		ApplicationPKIXCMP,
		ApplicationPostScript,
		ApplicationProtobuf,
		ApplicationProtobufJSON,
		ApplicationRDF,
		ApplicationSchemaJSON,
		ApplicationSessionInfo,
		ApplicationSetPayment,
		ApplicationSetPaymentInitiation,
		ApplicationSetRegistration,
		ApplicationSetRegistrationInitiation,
		ApplicationSGML,
		ApplicationSGMLOpenCatalog,
		ApplicationSOAPFastInfoset,
		ApplicationSOAP,
		ApplicationSparQL,
		ApplicationSparQLResults,
		ApplicationSysLogMsg,
		ApplicationTOML,
		ApplicationVCardJSON,
		ApplicationArrowFile,
		ApplicationArrowStream,
		ApplicationParquet,
		ApplicationThriftBinary,
		ApplicationThriftCompact,
		ApplicationThriftJSON,
		ApplicationVCardXML,
		ApplicationBZip3,
		ApplicationMaxmind,
		ApplicationMermaid,
		ApplicationMSExcel,
		ApplicationMSHTMLHelp,
		ApplicationMSPowerPoint,
		ApplicationMSWorks,
		ApplicationMsgPack,
		ApplicationSQLite3,
		ApplicationWASM,
		ApplicationXPKIMessage,
		ApplicationXWWWFormURLEncoded,
		ApplicationX509CACert,
		ApplicationX509CARACert,
		ApplicationX509NextCACert,
		ApplicationXML,
		ApplicationXHTML,
		ApplicationXSLT,
		ApplicationYAML,
		ApplicationZip,
		ApplicationZLib,
		ApplicationZStd,
	}
	audioTypes = []Type{
		Audio3GPP,
		Audio3GPP2,
		AudioAAC,
		AudioAC3,
		AudioAMR,
		AudioBasic,
		AudioClearMode,
		AudioCN,
		AudioDAT12,
		AudioDLS,
		AudioDV,
		AudioDVI4,
		AudioEVRC,
		AudioEVS,
		AudioExample,
		AudioFLAC,
		AudioFlexFEC,
		AudioMIDIClip,
		AudioMobileXMF,
		AudioMPA,
		AudioMP4,
		AudioMP4ALATM,
		AudioMPARobust,
		AudioMPEG,
		AudioMPEGGeneric,
		AudioOGG,
		AudioOpus,
		AudioParityFEC,
		AudioPCMA,
		AudioRTPLoopback,
		AudioRTPMIDI,
		AudioRTX,
		AudioSofa,
		AudioSoundFont,
		AudioSPMIDI,
		AudioSpeex,
		AudioVDVI,
	}
	exampleTypes = []Type{
		ExampleExample,
	}
	fontTypes = []Type{
		FontCollection,
		FontOTF,
		FontSFNT,
		FontTTF,
		FontWOFF,
		FontWOFF2,
	}
	hapticsTypes = []Type{
		HapticsIVS,
		HapticsHJIF,
		HapticsHMPG,
	}
	imageTypes = []Type{
		ImageAces,
		ImageAPNG,
		ImageAVCI,
		ImageAVCS,
		ImageAVIF,
		ImageBMP,
		ImageCGM,
		ImageExample,
		ImageGIF,
		ImageHEIC,
		ImageHEICSequence,
		ImageHEIF,
		ImageHEIFSequence,
		ImageJPEG,
		ImagePNG,
		ImagePWGRaster,
		ImageSVG,
		ImageTIFF,
		ImageTIFFFx,
		ImagePhotoshop,
		ImageClip,
		ImageIcon,
		ImageMozillaAPNG,
		ImageWebP,
		ImageWMF,
	}
	messageTypes = []Type{
		MessageBHTTP,
		MessageCPIM,
		MessageDeliveryStatus,
		MessageDispositionNotification,
		MessageExample,
		MessageExternalBody,
		MessageFeedbackReport,
		MessageGlobal,
		MessageGlobalDeliveryStatus,
		MessageGlobalDispositionNotification,
		MessageGlobalHeaders,
		MessageHTTP,
		MessageIMDN,
		MessageMLS,
		MessageOHTTPChunkedReq,
		MessageOHTTPChunkedRes,
		MessageOHTTPReq,
		MessageOHTTPRes,
		MessagePartial,
		MessageRFC822,
		MessageSIP,
		MessageSIPFrag,
		MessageTrackingStatus,
	}
	modelTypes = []Type{
		Model3MF,
		ModelE57,
		ModelExample,
		ModelGLTF,
		ModelGLTFJSON,
		ModelJT,
		ModelIGES,
		ModelMesh,
		ModelMTL,
		ModelObj,
		ModelPRC,
		ModelStep,
		ModelStepXML,
		ModelStepZIP,
		ModelStepXMLZip,
		ModelSTL,
		ModelU3D,
		ModelVRML,
		ModelX3D,
	}
	multipartTypes = []Type{
		MultipartAlternative,
		MultipartAppleDouble,
		MultipartByteRanges,
		MultipartDigest,
		MultipartEncrypted,
		MultipartExample,
		MultipartFormData,
		MultipartHeaderSet,
		MultipartMixed,
		MultipartMultilingual,
		MultipartParallel,
		MultipartRelated,
		MultipartReport,
		MultipartSigned,
		MultipartVoiceMessage,
		MultipartXMixedReplace,
	}
	textTypes = []Type{
		TextCacheManifest,
		TextCalendar,
		TextCQL,
		TextCQLExpression,
		TextCQLIdentifier,
		TextCSS,
		TextCSV,
		TextCSVSchema,
		TextDNS,
		TextEnriched,
		TextExample,
		TextHTML,
		TextJavaScript,
		TextMarkdown,
		TextMizar,
		TextN3,
		TextOrg,
		TextParameters,
		TextPlain,
		TextRFC822Headers,
		TextRichText,
		TextRTF,
		TextRTX,
		TextSGML,
		TextStrings,
		TextTabSeparatedValues,
		TextTurtle,
		TextVCard,
		TextGML,
		TextGraphVIZ,
		TextLatexZ,
		TextVTT,
		TextWGSL,
		TextXML,
		TextXMLExternalParsedEntity,
	}
	videoTypes = []Type{
		Video3GPP,
		Video3GPP2,
		Video3GPPTT,
		VideoAV1,
		VideoBMPEG,
		VideoBT656,
		VideoDV,
		VideoExample,
		VideoFFV1,
		VideoH261,
		VideoH263,
		VideoH264,
		VideoH265,
		VideoH266,
		VideoJPEG,
		VideoMP4,
		VideoMPV,
		VideoMPEG,
		VideoMPEG4Generic,
		VideoNV,
		VideoOGG,
		VideoPointer,
		VideoQuickTime,
		VideoRTPLoopback,
		VideoRTX,
		VideoCCTV,
		VideoYouTube,
		VideoVivo,
		VideoVP8,
		VideoVP9,
	}
)

func TestApplicationTypes(t *testing.T) {
	for _, at := range applicationTypes {
		assert.Ok(t, at.Validate(), "could not validate application type %q", at)
		assert.Equal(t, Application, at.Top(), "application type %q has incorrect top-level type", at)
		assert.NotEqual(t, at.Sub(), "", "application type %q has no subtype", at)
		assert.Equal(t, strings.ToLower(string(at)), string(at), "application type %q has uppercase characters", at)

		assert.True(t, at.IsApplication(), "application type %q is not an application type", at)
		assert.False(t, at.IsAudio(), "application type %q is an audio type", at)
		assert.False(t, at.IsExample(), "application type %q is an example type", at)
		assert.False(t, at.IsFont(), "application type %q is a font type", at)
		assert.False(t, at.IsHaptics(), "application type %q is a haptics type", at)
		assert.False(t, at.IsImage(), "application type %q is an image type", at)
		assert.False(t, at.IsMessage(), "application type %q is a message type", at)
		assert.False(t, at.IsModel(), "application type %q is a model type", at)
		assert.False(t, at.IsMultipart(), "application type %q is a multipart type", at)
		assert.False(t, at.IsText(), "application type %q is a text type", at)
		assert.False(t, at.IsVideo(), "application type %q is a video type", at)

	}
}

func TestAudioTypes(t *testing.T) {
	for _, at := range audioTypes {
		assert.Ok(t, at.Validate(), "could not validate audio type %q", at)
		assert.Equal(t, Audio, at.Top(), "audio type %q has incorrect top-level type", at)
		assert.NotEqual(t, at.Sub(), "", "audio type %q has no subtype", at)
		assert.Equal(t, strings.ToLower(string(at)), string(at), "audio type %q has uppercase characters", at)

		assert.True(t, at.IsAudio(), "audio type %q is not an audio type", at)
		assert.False(t, at.IsApplication(), "audio type %q is an application type", at)
		assert.False(t, at.IsExample(), "audio type %q is an example type", at)
		assert.False(t, at.IsFont(), "audio type %q is a font type", at)
		assert.False(t, at.IsHaptics(), "audio type %q is a haptics type", at)
		assert.False(t, at.IsImage(), "audio type %q is an image type", at)
		assert.False(t, at.IsMessage(), "audio type %q is a message type", at)
		assert.False(t, at.IsModel(), "audio type %q is a model type", at)
		assert.False(t, at.IsMultipart(), "audio type %q is a multipart type", at)
		assert.False(t, at.IsText(), "audio type %q is a text type", at)
		assert.False(t, at.IsVideo(), "audio type %q is a video type", at)
	}
}

func TestExampleTypes(t *testing.T) {
	for _, et := range exampleTypes {
		assert.Ok(t, et.Validate(), "could not validate example type %q", et)
		assert.Equal(t, Example, et.Top(), "example type %q has incorrect top-level type", et)
		assert.NotEqual(t, et.Sub(), "", "example type %q has no subtype", et)
		assert.Equal(t, strings.ToLower(string(et)), string(et), "example type %q has uppercase characters", et)

		assert.True(t, et.IsExample(), "example type %q is not an example type", et)
		assert.False(t, et.IsApplication(), "example type %q is an application type", et)
		assert.False(t, et.IsAudio(), "example type %q is an audio type", et)
		assert.False(t, et.IsFont(), "example type %q is a font type", et)
		assert.False(t, et.IsHaptics(), "example type %q is a haptics type", et)
		assert.False(t, et.IsImage(), "example type %q is an image type", et)
		assert.False(t, et.IsMessage(), "example type %q is a message type", et)
		assert.False(t, et.IsModel(), "example type %q is a model type", et)
		assert.False(t, et.IsMultipart(), "example type %q is a multipart type", et)
		assert.False(t, et.IsText(), "example type %q is a text type", et)
		assert.False(t, et.IsVideo(), "example type %q is a video type", et)
	}
}

func TestFontTypes(t *testing.T) {
	for _, ft := range fontTypes {
		assert.Ok(t, ft.Validate(), "could not validate font type %q", ft)
		assert.Equal(t, Font, ft.Top(), "font type %q has incorrect top-level type", ft)
		assert.NotEqual(t, ft.Sub(), "", "font type %q has no subtype", ft)
		assert.Equal(t, strings.ToLower(string(ft)), string(ft), "font type %q has uppercase characters", ft)

		assert.True(t, ft.IsFont(), "font type %q is not a font type", ft)
		assert.False(t, ft.IsApplication(), "font type %q is an application type", ft)
		assert.False(t, ft.IsAudio(), "font type %q is an audio type", ft)
		assert.False(t, ft.IsExample(), "font type %q is an example type", ft)
		assert.False(t, ft.IsHaptics(), "font type %q is a haptics type", ft)
		assert.False(t, ft.IsImage(), "font type %q is an image type", ft)
		assert.False(t, ft.IsMessage(), "font type %q is a message type", ft)
		assert.False(t, ft.IsModel(), "font type %q is a model type", ft)
		assert.False(t, ft.IsMultipart(), "font type %q is a multipart type", ft)
		assert.False(t, ft.IsText(), "font type %q is a text type", ft)
		assert.False(t, ft.IsVideo(), "font type %q is a video type", ft)
	}
}

func TestHapticsTypes(t *testing.T) {
	for _, ht := range hapticsTypes {
		assert.Ok(t, ht.Validate(), "could not validate haptics type %q", ht)
		assert.Equal(t, Haptics, ht.Top(), "haptics type %q has incorrect top-level type", ht)
		assert.NotEqual(t, ht.Sub(), "", "haptics type %q has no subtype", ht)
		assert.Equal(t, strings.ToLower(string(ht)), string(ht), "haptics type %q has uppercase characters", ht)

		assert.True(t, ht.IsHaptics(), "haptics type %q is not a haptics type", ht)
		assert.False(t, ht.IsApplication(), "haptics type %q is an application type", ht)
		assert.False(t, ht.IsAudio(), "haptics type %q is an audio type", ht)
		assert.False(t, ht.IsExample(), "haptics type %q is an example type", ht)
		assert.False(t, ht.IsFont(), "haptics type %q is a font type", ht)
		assert.False(t, ht.IsImage(), "haptics type %q is an image type", ht)
		assert.False(t, ht.IsMessage(), "haptics type %q is a message type", ht)
		assert.False(t, ht.IsModel(), "haptics type %q is a model type", ht)
		assert.False(t, ht.IsMultipart(), "haptics type %q is a multipart type", ht)
		assert.False(t, ht.IsText(), "haptics type %q is a text type", ht)
		assert.False(t, ht.IsVideo(), "haptics type %q is a video type", ht)
	}
}

func TestImageTypes(t *testing.T) {
	for _, it := range imageTypes {
		assert.Ok(t, it.Validate(), "could not validate image type %q", it)
		assert.Equal(t, Image, it.Top(), "image type %q has incorrect top-level type", it)
		assert.NotEqual(t, it.Sub(), "", "image type %q has no subtype", it)
		assert.Equal(t, strings.ToLower(string(it)), string(it), "image type %q has uppercase characters", it)

		assert.True(t, it.IsImage(), "image type %q is not an image type", it)
		assert.False(t, it.IsApplication(), "image type %q is an application type", it)
		assert.False(t, it.IsAudio(), "image type %q is an audio type", it)
		assert.False(t, it.IsExample(), "image type %q is an example type", it)
		assert.False(t, it.IsFont(), "image type %q is a font type", it)
		assert.False(t, it.IsHaptics(), "image type %q is a haptics type", it)
		assert.False(t, it.IsMessage(), "image type %q is a message type", it)
		assert.False(t, it.IsModel(), "image type %q is a model type", it)
		assert.False(t, it.IsMultipart(), "image type %q is a multipart type", it)
		assert.False(t, it.IsText(), "image type %q is a text type", it)
		assert.False(t, it.IsVideo(), "image type %q is a video type", it)
	}
}

func TestMessageTypes(t *testing.T) {
	for _, mt := range messageTypes {
		assert.Ok(t, mt.Validate(), "could not validate message type %q", mt)
		assert.Equal(t, Message, mt.Top(), "message type %q has incorrect top-level type", mt)
		assert.NotEqual(t, mt.Sub(), "", "message type %q has no subtype", mt)
		assert.Equal(t, strings.ToLower(string(mt)), string(mt), "message type %q has uppercase characters", mt)

		assert.True(t, mt.IsMessage(), "message type %q is not a message type", mt)
		assert.False(t, mt.IsApplication(), "message type %q is an application type", mt)
		assert.False(t, mt.IsAudio(), "message type %q is an audio type", mt)
		assert.False(t, mt.IsExample(), "message type %q is an example type", mt)
		assert.False(t, mt.IsFont(), "message type %q is a font type", mt)
		assert.False(t, mt.IsHaptics(), "message type %q is a haptics type", mt)
		assert.False(t, mt.IsImage(), "message type %q is an image type", mt)
		assert.False(t, mt.IsModel(), "message type %q is a model type", mt)
		assert.False(t, mt.IsMultipart(), "message type %q is a multipart type", mt)
		assert.False(t, mt.IsText(), "message type %q is a text type", mt)
		assert.False(t, mt.IsVideo(), "message type %q is a video type", mt)
	}
}

func TestModelTypes(t *testing.T) {
	for _, mt := range modelTypes {
		assert.Ok(t, mt.Validate(), "could not validate model type %q", mt)
		assert.Equal(t, Model, mt.Top(), "model type %q has incorrect top-level type", mt)
		assert.NotEqual(t, mt.Sub(), "", "model type %q has no subtype", mt)
		assert.Equal(t, strings.ToLower(string(mt)), string(mt), "model type %q has uppercase characters", mt)

		assert.True(t, mt.IsModel(), "model type %q is not a model type", mt)
		assert.False(t, mt.IsApplication(), "model type %q is an application type", mt)
		assert.False(t, mt.IsAudio(), "model type %q is an audio type", mt)
		assert.False(t, mt.IsExample(), "model type %q is an example type", mt)
		assert.False(t, mt.IsFont(), "model type %q is a font type", mt)
		assert.False(t, mt.IsHaptics(), "model type %q is a haptics type", mt)
		assert.False(t, mt.IsImage(), "model type %q is an image type", mt)
		assert.False(t, mt.IsMessage(), "model type %q is a message type", mt)
		assert.False(t, mt.IsMultipart(), "model type %q is a multipart type", mt)
		assert.False(t, mt.IsText(), "model type %q is a text type", mt)
		assert.False(t, mt.IsVideo(), "model type %q is a video type", mt)
	}
}

func TestMultipartTypes(t *testing.T) {
	for _, mt := range multipartTypes {
		assert.Ok(t, mt.Validate(), "could not validate multipart type %q", mt)
		assert.Equal(t, Multipart, mt.Top(), "multipart type %q has incorrect top-level type", mt)
		assert.NotEqual(t, mt.Sub(), "", "multipart type %q has no subtype", mt)
		assert.Equal(t, strings.ToLower(string(mt)), string(mt), "multipart type %q has uppercase characters", mt)

		assert.True(t, mt.IsMultipart(), "multipart type %q is not a multipart type", mt)
		assert.False(t, mt.IsApplication(), "multipart type %q is an application type", mt)
		assert.False(t, mt.IsAudio(), "multipart type %q is an audio type", mt)
		assert.False(t, mt.IsExample(), "multipart type %q is an example type", mt)
		assert.False(t, mt.IsFont(), "multipart type %q is a font type", mt)
		assert.False(t, mt.IsHaptics(), "multipart type %q is a haptics type", mt)
		assert.False(t, mt.IsImage(), "multipart type %q is an image type", mt)
		assert.False(t, mt.IsMessage(), "multipart type %q is a message type", mt)
		assert.False(t, mt.IsModel(), "multipart type %q is a model type", mt)
		assert.False(t, mt.IsText(), "multipart type %q is a text type", mt)
		assert.False(t, mt.IsVideo(), "multipart type %q is a video type", mt)
	}
}

func TestTextTypes(t *testing.T) {
	for _, tt := range textTypes {
		assert.Ok(t, tt.Validate(), "could not validate text type %q", tt)
		assert.Equal(t, Text, tt.Top(), "text type %q has incorrect top-level type", tt)
		assert.NotEqual(t, tt.Sub(), "", "text type %q has no subtype", tt)
		assert.Equal(t, strings.ToLower(string(tt)), string(tt), "text type %q has uppercase characters", tt)

		assert.True(t, tt.IsText(), "text type %q is not a text type", tt)
		assert.False(t, tt.IsApplication(), "text type %q is an application type", tt)
		assert.False(t, tt.IsAudio(), "text type %q is an audio type", tt)
		assert.False(t, tt.IsExample(), "text type %q is an example type", tt)
		assert.False(t, tt.IsFont(), "text type %q is a font type", tt)
		assert.False(t, tt.IsHaptics(), "text type %q is a haptics type", tt)
		assert.False(t, tt.IsImage(), "text type %q is an image type", tt)
		assert.False(t, tt.IsMessage(), "text type %q is a message type", tt)
		assert.False(t, tt.IsModel(), "text type %q is a model type", tt)
		assert.False(t, tt.IsMultipart(), "text type %q is a multipart type", tt)
		assert.False(t, tt.IsVideo(), "text type %q is a video type", tt)
	}
}

func TestVideoTypes(t *testing.T) {
	for _, vt := range videoTypes {
		assert.Ok(t, vt.Validate(), "could not validate video type %q", vt)
		assert.Equal(t, Video, vt.Top(), "video type %q has incorrect top-level type", vt)
		assert.NotEqual(t, vt.Sub(), "", "video type %q has no subtype", vt)
		assert.Equal(t, strings.ToLower(string(vt)), string(vt), "video type %q has uppercase characters", vt)

		assert.True(t, vt.IsVideo(), "video type %q is not a video type", vt)
		assert.False(t, vt.IsApplication(), "video type %q is an application type", vt)
		assert.False(t, vt.IsAudio(), "video type %q is an audio type", vt)
		assert.False(t, vt.IsExample(), "video type %q is an example type", vt)
		assert.False(t, vt.IsFont(), "video type %q is a font type", vt)
		assert.False(t, vt.IsHaptics(), "video type %q is a haptics type", vt)
		assert.False(t, vt.IsImage(), "video type %q is an image type", vt)
		assert.False(t, vt.IsMessage(), "video type %q is a message type", vt)
		assert.False(t, vt.IsModel(), "video type %q is a model type", vt)
		assert.False(t, vt.IsMultipart(), "video type %q is a multipart type", vt)
		assert.False(t, vt.IsText(), "video type %q is a text type", vt)
	}
}

func TestUnknownMediaType(t *testing.T) {
	assert.Ok(t, UnknownMimeType.Validate(), "could not validate unknown media type")
	assert.NotEqual(t, UnknownMimeType.Top(), "", "unknown media type has incorrect top-level type")
	assert.NotEqual(t, UnknownMimeType.Sub(), "", "unknown media type has no subtype")
	assert.Equal(t, strings.ToLower(string(UnknownMimeType)), string(UnknownMimeType), "unknown media type has uppercase characters")
}
