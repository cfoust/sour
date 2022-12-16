#ifndef __CUBE_H__
#define __CUBE_H__

#define _FILE_OFFSET_BITS 64

#ifdef __GNUC__
#define gamma __gamma
#endif

#ifdef WIN32
#define _USE_MATH_DEFINES
#endif
#include <math.h>

#ifdef __GNUC__
#undef gamma
#endif

#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <ctype.h>
#include <stdarg.h>
#include <limits.h>
#include <assert.h>
#include <time.h>

#ifdef _WIN32
  #define WIN32_LEAN_AND_MEAN
  #ifdef _WIN32_WINNT
  #undef _WIN32_WINNT
  #endif
  #define _WIN32_WINNT 0x0500
  #include "windows.h"
  #ifndef _WINDOWS
    #define _WINDOWS
  #endif
  #ifndef __GNUC__
    #include <eh.h>
    #include <dbghelp.h>
  #endif
  #define ZLIB_DLL
#endif

#include "enet/enet.h"
#include <zlib.h>

#ifdef __sun__
#undef sun
#undef MAXNAMELEN
#ifdef queue
  #undef queue
#endif
#define queue __squeue
#endif

#include "tools.h"
#include "geom.h"
#include "ents.h"
#include "command.h"

#include "iengine.h"
#include "igame.h"
#include "../engine/world.h"

#endif

