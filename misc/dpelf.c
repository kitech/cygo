#include <stdio.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <libelf.h>
#include <gelf.h>

void
main(int argc, char **argv)
{
    Elf         *elf;
    Elf_Scn     *scn = NULL;
    GElf_Shdr   shdr;
    Elf_Data    *data;
    int         fd, ii, count;

    elf_version(EV_CURRENT);

    fd = open(argv[1], O_RDONLY);
    elf = elf_begin(fd, ELF_C_READ, NULL);

    printf("%d %p\n", fd, elf);
    while ((scn = elf_nextscn(elf, scn)) != NULL) {
        gelf_getshdr(scn, &shdr);
        if (shdr.sh_type == SHT_DYNSYM) {
            break;
        }
        continue;
        if (shdr.sh_type == SHT_SYMTAB) {
            /* found a symbol table, go print it. */
            printf("found SHT_SYMTAB\n");
            break;
        }
    }

    data = elf_getdata(scn, NULL);
    printf("%d %d\n", shdr.sh_size, shdr.sh_entsize);
    if (shdr.sh_entsize == 0) {
        count = 10;
    }else{
        count = shdr.sh_size / shdr.sh_entsize;
    }

    /* print the symbol names */
    for (ii = 0; ii < count; ++ii) {
        GElf_Sym sym;
        gelf_getsym(data, ii, &sym);
        printf("%s %d\n", elf_strptr(elf, shdr.sh_link, sym.st_name), sym.st_size);
    }
    elf_end(elf);
    close(fd);
}
