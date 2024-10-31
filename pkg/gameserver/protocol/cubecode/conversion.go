package cubecode

// conversion table: cubecode → unicode (i.e. using the cubecode code point as index)
// cubecode is a small subset of unicode containing selected characters from the Basic Latin, Latin-1 Supplement,
// Latin Extended-A and Cyrillic blocks, that can be represented in 8-bit space. characters included from the Basic
// Latin block (all characters except most control characters) keep their position in unicode. unused positions in
// the 8-bit space are filled up with letters from later Unicode blocks, resulting in interspersed Basic Latin and
// Latin-1 Supplement characters at the beginning of the conversion table.
// example: server sends a 2, cubeToUni[2] → Á
var cubeToUni = [256]rune{
	// Basic Latin (deliberately omitting most control characters)
	'\x00',
	// Latin-1 Supplement (selected letters)
	'À', 'Á', 'Â', 'Ã', 'Ä', 'Å', 'Æ',
	'Ç',
	// Basic Latin (cont.)
	'\t', '\n', '\v', '\f', '\r',
	// Latin-1 Supplement (cont.)
	'È', 'É', 'Ê', 'Ë',
	'Ì', 'Í', 'Î', 'Ï',
	'Ñ',
	'Ò', 'Ó', 'Ô', 'Õ', 'Ö', 'Ø',
	'Ù', 'Ú', 'Û',
	// Basic Latin (cont.)
	' ', '!', '"', '#', '$', '%', '&', '\'', '(', ')', '*', '+', ',', '-', '.', '/',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	':', ';', '<', '=', '>', '?', '@',
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'[', '\\', ']', '^', '_', '`',
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	'{', '|', '}', '~',
	// Latin-1 Supplement (cont.)
	'Ü',
	'Ý',
	'ß',
	'à', 'á', 'â', 'ã', 'ä', 'å', 'æ',
	'ç',
	'è', 'é', 'ê', 'ë',
	'ì', 'í', 'î', 'ï',
	'ñ',
	'ò', 'ó', 'ô', 'õ', 'ö', 'ø',
	'ù', 'ú', 'û', 'ü',
	'ý', 'ÿ',
	// Latin Extended-A (selected letters)
	'Ą', 'ą',
	'Ć', 'ć', 'Č', 'č',
	'Ď', 'ď',
	'Ę', 'ę', 'Ě', 'ě',
	'Ğ', 'ğ',
	'İ', 'ı',
	'Ł', 'ł',
	'Ń', 'ń', 'Ň', 'ň',
	'Ő', 'ő', 'Œ', 'œ',
	'Ř', 'ř',
	'Ś', 'ś', 'Ş', 'ş', 'Š', 'š',
	'Ť', 'ť',
	'Ů', 'ů', 'Ű', 'ű',
	'Ÿ',
	'Ź', 'ź', 'Ż', 'ż', 'Ž', 'ž',
	// Cyrillic (selected letters, deliberately omitting letters visually identical to characters in Basic Latin)
	'Є',
	'Б' /**/, 'Г', 'Д', 'Ж', 'З', 'И', 'Й' /**/, 'Л' /*     */, 'П' /**/, 'У', 'Ф', 'Ц', 'Ч', 'Ш', 'Щ', 'Ъ', 'Ы', 'Ь', 'Э', 'Ю', 'Я',
	'б', 'в', 'г', 'д', 'ж', 'з', 'и', 'й', 'к', 'л', 'м', 'н', 'п', 'т' /**/, 'ф', 'ц', 'ч', 'ш', 'щ', 'ъ', 'ы', 'ь', 'э', 'ю', 'я',
	'є',
	'Ґ', 'ґ',
}

func ToUnicode(cpoint int32) rune {
	if 0 <= cpoint && cpoint < 256 {
		return cubeToUni[cpoint]
	}
	return '�'
}

// conversion table: unicode → cubecode (i.e. using the unicode code point as key)
// reverse of cubeToUni.
// example: you want to send 'ø', uni2Cube['ø'] → 152, 152 should be encoded in the packet using PutInt().
var uniToCube = map[rune]int32{}

func init() {
	for cpoint, r := range cubeToUni {
		uniToCube[r] = int32(cpoint)
	}
}

func FromUnicode(r rune) int32 {
	return uniToCube[r]
}
