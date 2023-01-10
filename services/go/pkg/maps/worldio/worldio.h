#ifndef WORLD_H
#define WORLD_H

#define _FILE_OFFSET_BITS 64

#include "tools.h"
#include "engine.h"

void freeocta(cube *c);
cube *loadchildren_buf(void *p, size_t len, int size, int _mapversion);
size_t savec_buf(void *p, unsigned int len, cube *c, int size);

cube *getcubeindex(cube *c, int i);
void cube_setedge(cube *c, int i, uchar value);
void cube_settexture(cube *c, int i, ushort value);
cube *apply_messages(cube *c, int _worldsize, void *data, size_t len);

#endif
