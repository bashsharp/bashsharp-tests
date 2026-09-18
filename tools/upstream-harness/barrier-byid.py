"""by-ID comparison of two barrier manifest dirs: failing keys = (root, mode) across active-*-manifest.tsv + active-unclassified.tsv"""
import sys, glob, os, csv
def load(d):
    keys={}
    for f in glob.glob(os.path.join(d,'active-*-manifest.tsv'))+[os.path.join(d,'active-unclassified.tsv')]:
        if not os.path.exists(f): continue
        with open(f) as fh:
            r=csv.DictReader(fh, delimiter='\t')
            for row in r:
                keys[(row['root'],row['mode'])]=(os.path.basename(f), row.get('first_line',''))
    return keys
a,b=load(sys.argv[1]),load(sys.argv[2])
print(f"{sys.argv[1]}: {len(a)} failing keys; {sys.argv[2]}: {len(b)} failing keys")
only_a=sorted(set(a)-set(b)); only_b=sorted(set(b)-set(a))
print(f"fixed in B (fail only in A): {len(only_a)}"); [print("  -",k,a[k][1][:100]) for k in only_a]
print(f"new in B (fail only in B): {len(only_b)}"); [print("  +",k,b[k][1][:100]) for k in only_b]
flips=[k for k in set(a)&set(b) if a[k][1]!=b[k][1]]
print(f"same key, first_line differs: {len(flips)}"); [print("  ~",k,'|',a[k][1][:70],'->',b[k][1][:70]) for k in sorted(flips)]
