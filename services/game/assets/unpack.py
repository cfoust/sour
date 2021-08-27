import sys
import re

FILENAME = "[-0-9A-Za-z_\/.]+"
EXEC_REGEX = re.compile('exec "?({})"?'.format(FILENAME))
TEXTURE_REGEX = re.compile('\s*texture [0-9a-z] "?({})"?'.format(FILENAME))

def get_cfg_files(cfg):
    lines = open(cfg, 'r').readlines()

    files = [cfg]

    for line in lines:
        exec_match = EXEC_REGEX.match(line)
        if exec_match:
            recurse = exec_match.group(1)
            files += get_cfg_files(recurse)
            continue

        texture_match = TEXTURE_REGEX.match(line)
        if texture_match:
            files.append(texture_match.group(1))
            continue

    return files

if __name__ == "__main__":
    target = sys.argv[1]
    print(target)
    textures = get_cfg_files(target)

    for texture in textures:
        print('packages/{}'.format(texture))
