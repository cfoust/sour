#ifndef CRYPTO_H
#define CRYPTO_H

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

void genauthkey(const char *secret, void *priv, void *pub);
void *genchallenge(const char *pubstr, const char *seed, int seedlen, void * challengeout, void * answerout);
void freechallenge(void *answer);
bool checkchallenge(const char *answerstr, void *correct);

#endif
