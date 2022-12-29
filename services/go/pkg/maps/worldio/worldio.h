#ifndef WORLD_H
#define WORLD_H

#define _FILE_OFFSET_BITS 64

#include <math.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <ctype.h>
#include <stdarg.h>
#include <limits.h>
#include <assert.h>
#include <time.h>

#include "tools.h"
#include "engine.h"

cube *loadchildren_buf(void *p, size_t len);

#endif
