export enum CubeMessageType {
  N_CONNECT = 0,
  N_SERVINFO,
  N_WELCOME,
  N_INITCLIENT,
  N_POS,
  N_TEXT,
  N_SOUND,
  N_CDIS,
  N_SHOOT,
  N_EXPLODE,
  N_SUICIDE,
  N_DIED,
  N_DAMAGE,
  N_HITPUSH,
  N_SHOTFX,
  N_EXPLODEFX,
  N_TRYSPAWN,
  N_SPAWNSTATE,
  N_SPAWN,
  N_FORCEDEATH,
  N_GUNSELECT,
  N_TAUNT,
  N_MAPCHANGE,
  N_MAPVOTE,
  N_TEAMINFO,
  N_ITEMSPAWN,
  N_ITEMPICKUP,
  N_ITEMACC,
  N_TELEPORT,
  N_JUMPPAD,
  N_PING,
  N_PONG,
  N_CLIENTPING,
  N_TIMEUP,
  N_FORCEINTERMISSION,
  N_SERVMSG,
  N_ITEMLIST,
  N_RESUME,
  N_EDITMODE,
  N_EDITENT,
  N_EDITF,
  N_EDITT,
  N_EDITM,
  N_FLIP,
  N_COPY,
  N_PASTE,
  N_ROTATE,
  N_REPLACE,
  N_DELCUBE,
  N_REMIP,
  N_EDITVSLOT,
  N_UNDO,
  N_REDO,
  N_NEWMAP,
  N_GETMAP,
  N_SENDMAP,
  N_CLIPBOARD,
  N_EDITVAR,
  N_MASTERMODE,
  N_KICK,
  N_CLEARBANS,
  N_CURRENTMASTER,
  N_SPECTATOR,
  N_SETMASTER,
  N_SETTEAM,
  N_BASES,
  N_BASEINFO,
  N_BASESCORE,
  N_REPAMMO,
  N_BASEREGEN,
  N_ANNOUNCE,
  N_LISTDEMOS,
  N_SENDDEMOLIST,
  N_GETDEMO,
  N_SENDDEMO,
  N_DEMOPLAYBACK,
  N_RECORDDEMO,
  N_STOPDEMO,
  N_CLEARDEMOS,
  N_TAKEFLAG,
  N_RETURNFLAG,
  N_RESETFLAG,
  N_INVISFLAG,
  N_TRYDROPFLAG,
  N_DROPFLAG,
  N_SCOREFLAG,
  N_INITFLAGS,
  N_SAYTEAM,
  N_CLIENT,
  N_AUTHTRY,
  N_AUTHKICK,
  N_AUTHCHAL,
  N_AUTHANS,
  N_REQAUTH,
  N_PAUSEGAME,
  N_GAMESPEED,
  N_ADDBOT,
  N_DELBOT,
  N_INITAI,
  N_FROMAI,
  N_BOTLIMIT,
  N_BOTBALANCE,
  N_MAPCRC,
  N_CHECKMAPS,
  N_SWITCHNAME,
  N_SWITCHMODEL,
  N_SWITCHTEAM,
  N_INITTOKENS,
  N_TAKETOKEN,
  N_EXPIRETOKENS,
  N_DROPTOKENS,
  N_DEPOSITTOKENS,
  N_STEALTOKENS,
  N_SERVCMD,
  N_DEMOPACKET,
  NUMMSG,
}

const CUBE_TO_UNI = [
  // Basic Latin (deliberately omitting most control characters)
  '\x00',
  // Latin-1 Supplement (selected letters)
  'À',
  'Á',
  'Â',
  'Ã',
  'Ä',
  'Å',
  'Æ',
  'Ç',
  // Basic Latin (cont.)
  '\t',
  '\n',
  '\v',
  '\f',
  '\r',
  // Latin-1 Supplement (cont.)
  'È',
  'É',
  'Ê',
  'Ë',
  'Ì',
  'Í',
  'Î',
  'Ï',
  'Ñ',
  'Ò',
  'Ó',
  'Ô',
  'Õ',
  'Ö',
  'Ø',
  'Ù',
  'Ú',
  'Û',
  // Basic Latin (cont.)
  ' ',
  '!',
  '"',
  '#',
  '$',
  '%',
  '&',
  "'",
  '(',
  ')',
  '*',
  '+',
  ',',
  '-',
  '.',
  '/',
  '0',
  '1',
  '2',
  '3',
  '4',
  '5',
  '6',
  '7',
  '8',
  '9',
  ':',
  ';',
  '<',
  '=',
  '>',
  '?',
  '@',
  'A',
  'B',
  'C',
  'D',
  'E',
  'F',
  'G',
  'H',
  'I',
  'J',
  'K',
  'L',
  'M',
  'N',
  'O',
  'P',
  'Q',
  'R',
  'S',
  'T',
  'U',
  'V',
  'W',
  'X',
  'Y',
  'Z',
  '[',
  '\\',
  ']',
  '^',
  '_',
  '`',
  'a',
  'b',
  'c',
  'd',
  'e',
  'f',
  'g',
  'h',
  'i',
  'j',
  'k',
  'l',
  'm',
  'n',
  'o',
  'p',
  'q',
  'r',
  's',
  't',
  'u',
  'v',
  'w',
  'x',
  'y',
  'z',
  '{',
  '|',
  '}',
  '~',
  // Latin-1 Supplement (cont.)
  'Ü',
  'Ý',
  'ß',
  'à',
  'á',
  'â',
  'ã',
  'ä',
  'å',
  'æ',
  'ç',
  'è',
  'é',
  'ê',
  'ë',
  'ì',
  'í',
  'î',
  'ï',
  'ñ',
  'ò',
  'ó',
  'ô',
  'õ',
  'ö',
  'ø',
  'ù',
  'ú',
  'û',
  'ü',
  'ý',
  'ÿ',
  // Latin Extended-A (selected letters)
  'Ą',
  'ą',
  'Ć',
  'ć',
  'Č',
  'č',
  'Ď',
  'ď',
  'Ę',
  'ę',
  'Ě',
  'ě',
  'Ğ',
  'ğ',
  'İ',
  'ı',
  'Ł',
  'ł',
  'Ń',
  'ń',
  'Ň',
  'ň',
  'Ő',
  'ő',
  'Œ',
  'œ',
  'Ř',
  'ř',
  'Ś',
  'ś',
  'Ş',
  'ş',
  'Š',
  'š',
  'Ť',
  'ť',
  'Ů',
  'ů',
  'Ű',
  'ű',
  'Ÿ',
  'Ź',
  'ź',
  'Ż',
  'ż',
  'Ž',
  'ž',
  // Cyrillic (selected letters, deliberately omitting letters visually
  // identical to characters in Basic Latin)
  'Є',
  'Б' /**/,
  'Г',
  'Д',
  'Ж',
  'З',
  'И',
  'Й' /**/,
  'Л' /*     */,
  'П' /**/,
  'У',
  'Ф',
  'Ц',
  'Ч',
  'Ш',
  'Щ',
  'Ъ',
  'Ы',
  'Ь',
  'Э',
  'Ю',
  'Я',
  'б',
  'в',
  'г',
  'д',
  'ж',
  'з',
  'и',
  'й',
  'к',
  'л',
  'м',
  'н',
  'п',
  'т' /**/,
  'ф',
  'ц',
  'ч',
  'ш',
  'щ',
  'ъ',
  'ы',
  'ь',
  'э',
  'ю',
  'я',
  'є',
  'Ґ',
  'ґ',
]

export function cubeToUnicode(cpoint: number): string {
  if (0 <= cpoint && cpoint < 256) {
    return CUBE_TO_UNI[cpoint]
  }
  return '�'
}

// conversion table: unicode → cubecode (i.e. using the unicode code point as
// key) reverse of cubeToUni. example: you want to send 'ø', uni2Cube['ø'] →
// 152, 152 should be encoded in the packet using PutInt().
const UNI_TO_CUBE: Record<string, number> = {}

for (let i = 0; i < UNI_TO_CUBE.length; i++) {
  UNI_TO_CUBE[CUBE_TO_UNI[i]] = i
}

export function uniToCube(uni: string): number {
  return UNI_TO_CUBE[uni]
}

export type Packet = {
  data: Uint8Array
  offset: number
}

export function newPacket(data: Uint8Array): Packet {
  return { data, offset: 0 }
}

export function remaining(p: Packet): number {
  return p.data.length - p.offset
}

export function getByte(p: Packet): Maybe<number> {
  if (remaining(p) < 1) {
    return null
  }

  return p.data[p.offset++]
}

export function getInt(p: Packet): Maybe<number> {
  const b = getByte(p)
  if (b == null) return null

  switch (b) {
    case 0x80: {
      if (remaining(p) < 2) {
        return null
      }
      const a = getByte(p)
      const b = getByte(p)
      if (a == null || b == null) return null
      return a + (b << 8)
      break
    }
    case 0x81: {
      if (remaining(p) < 4) {
        return null
      }
      const a = getByte(p)
      const b = getByte(p)
      const c = getByte(p)
      const d = getByte(p)
      if (a == null || b == null || c == null || d == null) return null
      return (((a + b) << (8 + c)) << (16 + d)) << 24
      break
    }
    default:
      return b
      break
  }
}

export function getString(p: Packet): Maybe<string> {
  let s: string = ''
  while (true) {
    const cpoint = getInt(p)
    if (cpoint == null) {
      return null
    }
    if (cpoint === 0) {
      return s
    }
    s += cubeToUnicode(cpoint)
  }
  return s
}
