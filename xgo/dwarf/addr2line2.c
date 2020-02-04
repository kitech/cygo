#include <stdlib.h>
#include <stdio.h>
#include <stdbool.h>
#include <unistd.h>
#include <getopt.h>
#include <err.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <libdwarf/libdwarf.h>
#include <libdwarf/dwarf.h>
#include <libelf.h>

#include <execinfo.h>
#include <string.h>

// #include "cxbase.h"

#define MAX_ADDR_LEN 20
#define BATCHMODE_HEURISTIC 100

typedef struct flags {
    bool addresses;
    bool batchmode;

    bool force_batchmode;
    bool force_nobatchmode;
} flagsT;

typedef struct lookup_table {
    Dwarf_Line *table;
    Dwarf_Line_Context *ctxts;
    int cnt;
    Dwarf_Addr low;
    Dwarf_Addr high;
} lookup_tableT;

static void err_handler(Dwarf_Error err, Dwarf_Ptr errarg)
{
    errx(EXIT_FAILURE, "libdwarf error: %d %s0", dwarf_errno(err), dwarf_errmsg(err));
}

static bool pc_in_die(Dwarf_Debug dbg, Dwarf_Die die, Dwarf_Addr pc)
{
    int ret;
    Dwarf_Addr cu_lowpc = DW_DLV_BADADDR, cu_highpc;
    enum Dwarf_Form_Class highpc_cls;
    ret = dwarf_lowpc(die, &cu_lowpc, NULL);
    if (ret == DW_DLV_OK) {
        if (pc == cu_lowpc) {
            return true;
        }
        ret = dwarf_highpc_b(die, &cu_highpc, NULL, &highpc_cls, NULL);
        if (ret == DW_DLV_OK) {
            if (highpc_cls == DW_FORM_CLASS_CONSTANT) {
                cu_highpc += cu_lowpc;
            }
            if (pc >= cu_lowpc && pc < cu_highpc) {
                return true;
            }
        }
    }
    Dwarf_Attribute attr;
    if (dwarf_attr(die, DW_AT_ranges, &attr, NULL) == DW_DLV_OK) {
        Dwarf_Unsigned offset;
        if (dwarf_global_formref(attr, &offset, NULL) == DW_DLV_OK) {
            Dwarf_Signed count = 0;
            Dwarf_Ranges *ranges = 0;
            Dwarf_Addr baseaddr = 0;
            if (cu_lowpc != DW_DLV_BADADDR) {
                baseaddr = cu_lowpc;
            }
            ret = dwarf_get_ranges_a(dbg, offset, die,
                                     &ranges, &count, NULL, NULL);
            for(int i = 0; i < count; i++) {
                Dwarf_Ranges *cur = ranges + i;
                if (cur->dwr_type == DW_RANGES_ENTRY) {
                    Dwarf_Addr rng_lowpc, rng_highpc;
                    rng_lowpc = baseaddr + cur->dwr_addr1;
                    rng_highpc = baseaddr + cur->dwr_addr2;
                    if (pc >= rng_lowpc && pc < rng_highpc) {
                        dwarf_ranges_dealloc(dbg, ranges, count);
                        dwarf_dealloc(dbg, attr, DW_DLA_ATTR);
                        return true;
                    }
                } else if (cur->dwr_type == DW_RANGES_ADDRESS_SELECTION) {
                    baseaddr = cur->dwr_addr2;
                } else {  // DW_RANGES_END
                    baseaddr = cu_lowpc;
                }
            }
            dwarf_ranges_dealloc(dbg, ranges, count);
        }
        dwarf_dealloc(dbg, attr, DW_DLA_ATTR);
    }
    return false;
}

static void print_line(Dwarf_Debug dbg, flagsT *flags, Dwarf_Line line, Dwarf_Addr pc)
{
    char *linesrc;
    Dwarf_Unsigned lineno;
    if (flags->addresses) {
        printf("%#018" DW_PR_DUx "\n", pc);
    }
    if (line) {
        dwarf_linesrc(line, &linesrc, NULL);
        dwarf_lineno(line, &lineno, NULL);
    } else {
        linesrc = (char*)"??";
        lineno = 0;
    }
    // printf("%s:%" DW_PR_DUu " %d %p\n", linesrc, lineno, line, pc);
    if (line) {
        dwarf_dealloc(dbg, linesrc, DW_DLA_STRING);
    }
}
static void print_line2(Dwarf_Debug dbg, flagsT *flags, Dwarf_Line line, Dwarf_Addr pc,
                        char*file, int*lineno2)
{
    char *linesrc = 0;
    Dwarf_Unsigned lineno = 0;
    if (flags->addresses) {
        printf("%#018" DW_PR_DUx "\n", pc);
    }
    if (line) {
        Dwarf_Error err = 0;
        dwarf_linesrc(line, &linesrc, &err);
        dwarf_lineno(line, &lineno, &err);
    } else {
        linesrc = (char*)"??";
        lineno = 0;
    }
    // printf("%s:%" DW_PR_DUu " %d %p\n", linesrc, lineno, line, pc);
    if (linesrc != 0 && file != 0) {
        *lineno2 = lineno;
        strcpy(file, linesrc);
    }
    if (line) {
        dwarf_dealloc(dbg, linesrc, DW_DLA_STRING);
    }
}

static bool lookup_pc_cu(Dwarf_Debug dbg, flagsT *flags, Dwarf_Addr pc, Dwarf_Die cu_die,
                         char*file, int*lineno2)
{
    int ret;
    Dwarf_Unsigned version;
    Dwarf_Small table_count;
    Dwarf_Line_Context ctxt;
    ret = dwarf_srclines_b(cu_die, &version, &table_count, &ctxt, NULL);
    if (ret == DW_DLV_NO_ENTRY) {
        return false;
    }
    bool is_found = false;
    if (table_count == 1) {
        Dwarf_Line *linebuf = 0;
        Dwarf_Signed linecount = 0;
        Dwarf_Error err;
        ret = dwarf_srclines_from_linecontext(ctxt, &linebuf, &linecount, &err);
        if (ret == DW_DLV_ERROR) {
            dwarf_srclines_dealloc_b(ctxt);
            err_handler(err, NULL);
        }
        Dwarf_Addr prev_lineaddr;
        Dwarf_Line prev_line = 0;
        for (int i = 0; i < linecount; i++) {
            Dwarf_Line line = linebuf[i];
            Dwarf_Addr lineaddr;
            dwarf_lineaddr(line, &lineaddr, NULL);
            if (pc == lineaddr) {
                /* Print the last line entry containing current pc. */
                Dwarf_Line last_pc_line = line;
                for (int j = i + 1; j < linecount; j++) {
                    Dwarf_Line j_line = linebuf[j];
                    dwarf_lineaddr(j_line, &lineaddr, NULL);
                    if (pc == lineaddr) {
                        last_pc_line = j_line;
                    }
                }
                is_found = true;
                print_line(dbg, flags, last_pc_line, pc);
                print_line2(dbg, flags, last_pc_line, pc, file, lineno2);
                break;
            } else if (prev_line && pc > prev_lineaddr && pc < lineaddr) {
                is_found = true;
                print_line(dbg, flags, prev_line, pc);
                print_line2(dbg, flags, prev_line, pc, file, lineno2);
                break;
            }
            Dwarf_Bool is_lne;
            dwarf_lineendsequence(line, &is_lne, NULL);
            if (is_lne) {
                prev_line = 0;
            } else {
                prev_lineaddr = lineaddr;
                prev_line = line;
            }
        }
    }
    dwarf_srclines_dealloc_b(ctxt);
    return is_found;
}

// return prev_line for call print_line
static bool lookup_pc(Dwarf_Debug dbg, flagsT *flags, Dwarf_Addr pc, char*file, int*lineno2)
{
    Dwarf_Bool is_info = true;
    Dwarf_Unsigned next_cu_header;
    Dwarf_Half header_cu_type;
    int ret;
    int cu_i;
    for (int cu_i = 0;; cu_i++) {
        ret = dwarf_next_cu_header_d(dbg, is_info, NULL, NULL, NULL, NULL, NULL, NULL,
                                     NULL, NULL, &next_cu_header, &header_cu_type, NULL);
        if (ret == DW_DLV_NO_ENTRY) {
            break;
        }
        Dwarf_Die cu_die = 0;
        ret = dwarf_siblingof_b(dbg, 0, is_info, &cu_die, NULL);
        if (ret == DW_DLV_OK) {
            if (pc_in_die(dbg, cu_die, pc)) {
                bool lookup_ret = lookup_pc_cu(dbg, flags, pc, cu_die, file, lineno2);
                dwarf_dealloc(dbg, cu_die, DW_DLA_DIE);
                while (dwarf_next_cu_header_d(dbg, is_info, NULL, NULL, NULL, NULL, NULL, NULL,
                                              NULL, NULL, &next_cu_header, &header_cu_type, NULL)
                       != DW_DLV_NO_ENTRY) {}
                return lookup_ret;
            } else {
                dwarf_dealloc(dbg, cu_die, DW_DLA_DIE);
                cu_die = 0;
            }
        }
    }
    return false;
}

static bool get_pc_range_die(Dwarf_Die die, Dwarf_Addr *low_out, Dwarf_Addr *high_out)
{
    int ret;
    Dwarf_Addr lowpc, highpc;
    bool is_low_set = false, is_high_set = false;
    enum Dwarf_Form_Class highpc_cls;
    ret = dwarf_lowpc(die, &lowpc, NULL);
    if (ret == DW_DLV_OK) {
        is_low_set = true;
        *low_out = lowpc;
    }
    if (is_low_set) {
        ret = dwarf_highpc_b(die, &highpc, NULL, &highpc_cls, NULL);
        if (ret == DW_DLV_OK) {
            if (highpc_cls == DW_FORM_CLASS_CONSTANT) {
                *high_out = lowpc + highpc;
            } else {
                *high_out = highpc;
            }
            is_high_set = true;
        }
    }
    return is_low_set && is_high_set;
}

static bool get_pc_range(Dwarf_Debug dbg, Dwarf_Addr *lowest, Dwarf_Addr *highest, int *cu_cnt)
{
    Dwarf_Bool is_info = true;
    Dwarf_Unsigned next_cu_header;
    Dwarf_Half header_cu_type;
    *lowest = DW_DLV_BADADDR;
    *highest = 0;
    int ret, cu_i;
    for (cu_i = 0;; cu_i++) {
        ret = dwarf_next_cu_header_d(dbg, is_info, NULL, NULL, NULL, NULL, NULL, NULL,
                                     NULL, NULL, &next_cu_header, &header_cu_type, NULL);
        if (ret == DW_DLV_NO_ENTRY) {
            break;
        }
        Dwarf_Die cu_die = 0;
        ret = dwarf_siblingof_b(dbg, 0, is_info, &cu_die, NULL);
        if (ret == DW_DLV_OK) {
            Dwarf_Addr cu_lowpc = DW_DLV_BADADDR, cu_highpc;
            enum Dwarf_Form_Class highpc_cls;
            ret = dwarf_lowpc(cu_die, &cu_lowpc, NULL);
            if (ret == DW_DLV_OK) {
                if (cu_lowpc < *lowest) {
                    *lowest = cu_lowpc;
                }
                ret = dwarf_highpc_b(cu_die, &cu_highpc, NULL, &highpc_cls, NULL);
                if (ret == DW_DLV_OK) {
                    if (highpc_cls == DW_FORM_CLASS_CONSTANT) {
                        cu_highpc += cu_lowpc;
                    }
                    if (cu_highpc > *highest) {
                        *highest = cu_highpc;
                    }
                }
            }
            Dwarf_Attribute attr;
            if (dwarf_attr(cu_die, DW_AT_ranges, &attr, NULL) == DW_DLV_OK) {
                Dwarf_Unsigned offset;
                if (dwarf_global_formref(attr, &offset, NULL) == DW_DLV_OK) {
                    Dwarf_Signed count = 0;
                    Dwarf_Ranges *ranges = 0;
                    Dwarf_Addr baseaddr = 0;
                    if (cu_lowpc != DW_DLV_BADADDR) {
                        baseaddr = cu_lowpc;
                    }
                    ret = dwarf_get_ranges_a(dbg, offset, cu_die,
                                             &ranges, &count, NULL, NULL);
                    for(int i = 0; i < count; i++) {
                        Dwarf_Ranges *cur = ranges + i;
                        if (cur->dwr_type == DW_RANGES_ENTRY) {
                            Dwarf_Addr rng_lowpc, rng_highpc;
                            rng_lowpc = baseaddr + cur->dwr_addr1;
                            rng_highpc = baseaddr + cur->dwr_addr2;
                            if (rng_lowpc < *lowest) {
                                *lowest = rng_lowpc;
                            }
                            if (rng_highpc > *highest) {
                                *highest = rng_highpc;
                            }
                        } else if (cur->dwr_type == DW_RANGES_ADDRESS_SELECTION) {
                            baseaddr = cur->dwr_addr2;
                        } else {  // DW_RANGES_END
                            baseaddr = cu_lowpc;
                        }
                    }
                    dwarf_ranges_dealloc(dbg, ranges, count);
                }
                dwarf_dealloc(dbg, attr, DW_DLA_ATTR);
            }
            dwarf_dealloc(dbg, cu_die, DW_DLA_DIE);
            cu_die = 0;
        }
    }
    *cu_cnt = cu_i;
    return (*lowest != DW_DLV_BADADDR && *highest != 0);
}

static void populate_lookup_table_die(Dwarf_Debug dbg, lookup_tableT *lookup_table, int cu_i, Dwarf_Die cu_die)
{
    int ret;
    Dwarf_Unsigned version;
    Dwarf_Small table_count;
    ret = dwarf_srclines_b(cu_die, &version, &table_count, &lookup_table->ctxts[cu_i], NULL);
    if (ret == DW_DLV_NO_ENTRY) {
        return;
    }
    bool is_found = false;
    if (table_count == 1) {
        Dwarf_Line *linebuf = 0;
        Dwarf_Signed linecount = 0;
        Dwarf_Error err;
        ret = dwarf_srclines_from_linecontext(lookup_table->ctxts[cu_i], &linebuf, &linecount, &err);
        if (ret == DW_DLV_ERROR) {
            dwarf_srclines_dealloc_b(lookup_table->ctxts[cu_i]);
            err_handler(err, NULL);
        }
        Dwarf_Addr prev_lineaddr;
        Dwarf_Line prev_line = 0;
        for (int i = 0; i < linecount; i++) {
            Dwarf_Line line = linebuf[i];
            Dwarf_Addr lineaddr;
            dwarf_lineaddr(line, &lineaddr, NULL);
            if (prev_line) {
                for (Dwarf_Addr addr = prev_lineaddr; addr < lineaddr; addr++) {
                    lookup_table->table[addr - lookup_table->low] = linebuf[i - 1];
                }
            }
            Dwarf_Bool is_lne;
            dwarf_lineendsequence(line, &is_lne, NULL);
            if (is_lne) {
                prev_line = 0;
            } else {
                prev_lineaddr = lineaddr;
                prev_line = line;
            }
        }
    }
}

static void populate_lookup_table(Dwarf_Debug dbg, lookup_tableT *lookup_table)
{
    Dwarf_Bool is_info = true;
    Dwarf_Unsigned next_cu_header;
    Dwarf_Half header_cu_type;
    int ret;
    for (int cu_i = 0;; cu_i++) {
        ret = dwarf_next_cu_header_d(dbg, is_info, NULL, NULL, NULL, NULL, NULL, NULL,
                                     NULL, NULL, &next_cu_header, &header_cu_type, NULL);
        if (ret == DW_DLV_NO_ENTRY) {
            break;
        }
        Dwarf_Die cu_die = 0;
        ret = dwarf_siblingof_b(dbg, 0, is_info, &cu_die, NULL);
        if (ret == DW_DLV_OK) {
            populate_lookup_table_die(dbg, lookup_table, cu_i, cu_die);
            dwarf_dealloc(dbg, cu_die, DW_DLA_DIE);
            cu_die = 0;
        }
    }
}

static int create_lookup_table(Dwarf_Debug dbg, lookup_tableT *lookup_table)
{
    Dwarf_Addr low, high;
    int cu_cnt;
    int result = 0;
    if (! get_pc_range(dbg, &low, &high, &cu_cnt)) {
        goto exit;
    }
    lookup_table->table = (Dwarf_Line**)malloc((high - low) * sizeof(Dwarf_Line));
    if (! lookup_table->table) {
        goto exit;
    }
    lookup_table->ctxts = (Dwarf_Line_Context**)malloc(cu_cnt * sizeof(Dwarf_Line_Context));
    if (! lookup_table->ctxts) {
        goto free_table_exit;
    }
    lookup_table->cnt = cu_cnt;
    lookup_table->low = low;
    lookup_table->high = high;
    populate_lookup_table(dbg, lookup_table);
    return 0;
 free_table_exit:
    free(lookup_table->table);
 exit:
    lookup_table->table = NULL;
    lookup_table->ctxts = NULL;
    return 1;
}

static void delete_lookup_table(lookup_tableT *lookup_table)
{
    free(lookup_table->table);
    lookup_table->table = NULL;
    for (int i = 0; i < lookup_table->cnt; i++) {
        dwarf_srclines_dealloc_b(lookup_table->ctxts[i]);
    }
    free(lookup_table->ctxts);
    lookup_table->ctxts = NULL;
}

static char *get_pc_buf(int argc, char **argv, char *buf, bool do_read_stdin)
{
    if (do_read_stdin) {
        return fgets(buf, MAX_ADDR_LEN, stdin);
    } else {
        if (optind < argc) {
            return argv[optind++];
        } else {
            return NULL;
        }
    }
}

static void populate_options(int argc, char *argv[], char **objfile, flagsT *flags)
{
    int c;
    while (1) {
        int this_option_optind = optind ? optind : 1;
        int option_index = 0;
        static struct option longopts[] =
            {
             {"addresses", no_argument, 0, 'a'},
             {"exe", required_argument, 0, 'e'},
             {"force-batch", no_argument, 0, 'b'},
             {"force-no-batch", no_argument, 0, 'n'},
             {0, 0, 0, 0}
            };
        c = getopt_long(argc, argv, "ae:bn", longopts, &option_index);
        if (c == -1) {
            break;
        }
        switch (c) {
        case 'a':
            flags->addresses = true;
            break;
        case 'e':
            *objfile = optarg;
            break;
        case 'b':
            flags->force_batchmode = true;
            break;
        case 'n':
            flags->force_nobatchmode = true;
            break;
        case '?':
            break;
        default:
            printf("?? getopt returned character code 0%o ??\n", c);
            break;
        }
    }
}

int main_rtdebug2(int argc, char *argv[])
{
    flagsT flags = {0};
    char *objfile = (char*)"a.out";
    populate_options(argc, argv, &objfile, &flags);

    int ret;
    Dwarf_Debug dbg;
    ret = dwarf_init_path(objfile, NULL, 0, DW_DLC_READ, DW_GROUPNUMBER_ANY, err_handler, NULL, &dbg, 0, 0, 0, NULL);
    if (ret == DW_DLV_NO_ENTRY) {
        errx(EXIT_FAILURE, "%s not found", objfile);
    }

    bool do_read_stdin = (optind >= argc);
    if (! flags.force_nobatchmode && (flags.force_batchmode || do_read_stdin || (argc + BATCHMODE_HEURISTIC) > optind)) {
        flags.batchmode = true;
    }
    lookup_tableT lookup_table;
    if (flags.batchmode) {
        create_lookup_table(dbg, &lookup_table);
    }

    char buf[MAX_ADDR_LEN], *pc_buf, *endptr;
    while ((pc_buf = get_pc_buf(argc, argv, buf, do_read_stdin))) {
        Dwarf_Addr pc = strtoull(pc_buf, &endptr, 16);
        bool is_found = false;
        if (endptr != pc_buf) {
            if (flags.batchmode && lookup_table.table) {
                if (pc >= lookup_table.low && pc < lookup_table.high) {
                    Dwarf_Line line = lookup_table.table[pc - lookup_table.low];
                    if (line) {
                        print_line(dbg, &flags, line, pc);
                        is_found = true;
                    }
                }
            } else {
                is_found = lookup_pc(dbg, &flags, pc, 0, 0);
            }
        }
        if (! is_found) {
            print_line(dbg, &flags, NULL, pc);
        }
    }
    if (flags.batchmode && lookup_table.table) {
        delete_lookup_table(&lookup_table);
    }
    dwarf_finish(dbg, NULL);
    return 0;
}

#include <unistd.h>
extern char __executable_start;
extern char __etext;
extern char** cxargv;
struct rtinfo2 {
    // public:
    int inited;
    int exefd;
    Dwarf_Debug dbg;
    Dwarf_Error err;
    lookup_tableT lookup_table;
    void* elf_start_addr; // objdump -f ./execute
    void* process_base_virtaddr; // _start address
    void* process_main_virtaddr; // main address
    void* process_stack_virtaddr; // __libc_start_main address
};
typedef struct rtinfo2 rtinfo2;

rtinfo2 rti2 = {.inited=0, .exefd=-1};

void get_elf_start_addr()  {
    void* exeadr = &__executable_start;
    void* etxtadr = &__etext;
    // printf("0x%lx\n", (unsigned long)&__executable_start);
    // printf("0x%lx\n", (unsigned long)&__etext);
    rti2.elf_start_addr = exeadr;
    // rti2.elf_start_addr = etxtadr;
}

void get_process_start_virt_addr()  {
    char *buf[100] = {0};
    int nptr = backtrace((void**)buf, 100);
    u64 ivaddr = 0;
    char** stklst = backtrace_symbols((void**)buf, nptr);
    for (int i =0; i < nptr; i++) {
        char* lbrackpos = strchr(stklst[i], '[');
        char* rbrackpos = strchr(stklst[i], ']');
        char adrbuf[32] = {0};
        lbrackpos++;
        memcpy(adrbuf, lbrackpos, rbrackpos-lbrackpos);
        ivaddr = strtoul(adrbuf, 0, 16);
        // printf("%d %s %p\n", i, stklst[i], &nptr);
        // printf("%d %s %p\n", i, adrbuf, &nptr);
        if (strstr(stklst[i], "(_start+")) {
            rti2.process_base_virtaddr = (void*)ivaddr;
        }
        else if (strstr(stklst[i], "(__libc_start_main+")) {
            rti2.process_stack_virtaddr = (void*)ivaddr;
        }
        else if (strstr(stklst[i], "(main+")) {
            rti2.process_main_virtaddr = (void*)ivaddr;
        }
    }
    free(stklst);
}

int init_elf_dwarf2()
{
    flagsT flags = {0};
    const char *objfile = cxargv[0];
    // populate_options(argc, argv, &objfile, &flags);

    int ret;
    Dwarf_Debug dbg;
    ret = dwarf_init_path(objfile, NULL, 0, DW_DLC_READ, DW_GROUPNUMBER_ANY, err_handler, NULL, &dbg, 0, 0, 0, NULL);
    if (ret == DW_DLV_NO_ENTRY) {
        errx(EXIT_FAILURE, "%s not found", objfile);
    }
    rti2.dbg = dbg;
    create_lookup_table(dbg, &rti2.lookup_table);

    get_elf_start_addr();
    get_process_start_virt_addr();
    void* pbaddr = rti2.process_base_virtaddr;
    void* stkaddr = rti2.process_stack_virtaddr;
    void* mainaddr = rti2.process_main_virtaddr;

    char tstadr[100] = {0};
    sprintf(tstadr, "%p",  &ret);
    // printf("t12345 %s\n", tstadr);
    Dwarf_Addr pc = strtoull(tstadr, NULL, 16);
    pc = 0xed5c;
    extern int main(int argc, char**argv);
    voidptr tptr = (void*)&init_elf_dwarf2;
    pc =   (char*)tptr-(char*)mainaddr ;
    pc += 0x7520;
    // printf("ddd %p %p %p-%p\n", pc , rti2.lookup_table.low, pbaddr, tptr);
    // Dwarf_Line line = rti2.lookup_table.table[pc - rti2.lookup_table.low];
    // if (line) {
    // print_line(dbg, &flags, line, pc);
        // is_found = true;
    // }
    print_line(dbg, &flags, NULL, pc);
    bool rv = lookup_pc(dbg, &flags, pc, 0, 0);
    // printf("%d\n", rv);

    return 0;
}

void rtdebug2_addr2line(void*ptr, char* file, int* lineno) {
    flagsT flags = {0};
    Dwarf_Addr pc = (Dwarf_Addr)ptr;
    pc = (char*)ptr - (char*)rti2.process_base_virtaddr;
    pc = (char*)ptr - (char*)rti2.elf_start_addr;
    pc -= 0x8; // magic
    // pc += 0x7545;
    // pc = 0x92a8;
    *lineno = 0;
    print_line2(rti2.dbg, &flags, NULL, pc, file, lineno);
    // printf("a2l 222 rv=%d %p %s %d\n", 0, pc, file, *lineno);
    if (*lineno == 0) {
        bool prevline = lookup_pc(rti2.dbg, &flags, pc, file, lineno);
        // printf("a2l 222 rv=%d %p %s %d\n", prevline, pc, file, *lineno);
        // print_line2(rti2.dbg, &flags, prevline, pc, file, lineno);
        // printf("a2l 222 rv=%d %p %s %d\n", prevline, pc, file, *lineno);
    }
}

// https://github.com/Crablicious/libdwarf-addr2line/blob/master/addr2line.c

