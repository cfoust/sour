#ifndef WORLD_H
#define WORLD_H

#define _FILE_OFFSET_BITS 64

#include "tools.h"
#include "engine.h"

void freeocta(cube *c);
cube *loadchildren_buf(void *p, size_t len, int size);
size_t savec_buf(void *p, size_t len, cube *c, int size);

#endif
