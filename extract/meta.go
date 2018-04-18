package extract

var meta_files = map[string]bool{
	"$MFT":     true,
	"$MFTMirr": true,
	"$LogFile": true,
	"$Volume":  true,
	"$AttrDef": true,
	"$Bitmap":  true,
	"$Boot":    true,
	"$BadClus": true,
	"$Quota":   true,
	"$Secure":  true,
	"$UpCase":  true,
	"$Extend":  true,
	"$ObjId":   true,
	"$Reparse": true,
	"$UsnJrnl": true,
}

func IsMetaFile(file *File) bool {
	if file == nil {
		return false
	}

	return meta_files[file.Name]
}
