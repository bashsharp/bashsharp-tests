# Grammar oracle comparison

Toolchain: local go1.27.2 (cached toolchain; host review status unverified). Reference: sh adapter.

Gate: **FAIL**. Fixtures: 521.

| Lane / class | Total | Agree | Disagree | Missing | Error |
|---|---:|---:|---:|---:|---:|
| island / contextual | 1 | 1 | 0 | 0 | 0 |
| island / not-applicable | 3 | 0 | 0 | 0 | 0 |
| island / required-negative | 5 | 5 | 0 | 0 | 0 |
| island / required-positive | 26 | 25 | 1 | 0 | 0 |
| mixed / planned | 7 | 6 | 1 | 0 | 0 |
| mixed / required-negative | 108 | 108 | 0 | 0 | 0 |
| mixed / required-positive | 190 | 180 | 10 | 0 | 0 |
| unit / not-applicable | 77 | 0 | 0 | 0 | 0 |
| unit / required-positive | 184 | 184 | 0 | 0 | 0 |

## Category totals

| Category / outcome | Count |
|---|---:|
| boundary / disagree | 10 |
| semantic / agree | 1 |
| syntax / agree | 508 |
| syntax / disagree | 2 |

## Disagreements and incomplete results

- s6/M10: tests:tests/go-delta/M10.bsh at byte 38 (3:9); oracle accept [] boundary [{typed-func 0 104}]; reference reject "bash++ for condition must be a scalar expression" boundary []; outcome syntax/disagree.
- polyglot/cpp-error.bpp: tests:tools/polyglot-gate.sh#L158 at byte 93 (6:1); oracle reject "mismatched input '<EOF>' expecting {BLOCK_OPEN, W_IF, W_CASE, W_WHILE, W_UNTIL, W_FOR, W_SELECT, W_LBRACE, W_DLBRACK, ARITH_CMD, SH_LPAREN}" boundary []; reference accept [] boundary [{fence 0 85}]; outcome syntax/disagree.
- startsite/decorator-body-brace-group: tests:tools/startsites/shapes.tsv#decorator-body-brace-group at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 11}]; outcome boundary/disagree.
- startsite/decorator-body-subshell: tests:tools/startsites/shapes.tsv#decorator-body-subshell at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 10}]; outcome boundary/disagree.
- startsite/decorator-body-arithmetic: tests:tools/startsites/shapes.tsv#decorator-body-arithmetic at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 12}]; outcome boundary/disagree.
- startsite/decorator-body-conditional: tests:tools/startsites/shapes.tsv#decorator-body-conditional at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 15}]; outcome boundary/disagree.
- startsite/decorator-body-select: tests:tools/startsites/shapes.tsv#decorator-body-select at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 32}]; outcome boundary/disagree.
- startsite/decorator-body-for: tests:tools/startsites/shapes.tsv#decorator-body-for at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 29}]; outcome boundary/disagree.
- startsite/decorator-body-case: tests:tools/startsites/shapes.tsv#decorator-body-case at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 26}]; outcome boundary/disagree.
- startsite/decorator-body-if: tests:tools/startsites/shapes.tsv#decorator-body-if at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 21}]; outcome boundary/disagree.
- startsite/decorator-body-while: tests:tools/startsites/shapes.tsv#decorator-body-while at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 24}]; outcome boundary/disagree.
- startsite/decorator-body-until: tests:tools/startsites/shapes.tsv#decorator-body-until at byte 0 (1:1); oracle accept [] boundary []; reference accept [] boundary [{at-function 0 24}]; outcome boundary/disagree.

## Semantic cases

- delta/reject/embed-missing.bsh: sh:grammar/testdata/reject/embed-missing.bsh at byte 14 (1:15); oracle accept []; reference reject "embed: open sh:grammar/testdata/reject/missing.py: no such file or directory"; production -.

## Planned and not applicable

- tour/basics.article#inline-01-L25 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 b8323d33054e451ebb746643f5820c295db743ca596d35da123131429f34a063); not a complete unit
- tour/basics.article#inline-02-L67 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 0cc3833e6bea4ba86361b74a2e2c3351e73958fa806a54acb50f88d383a4862a); not a complete unit
- tour/basics.article#inline-03-L71 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 b906fb6502ba6d21d2a38e6515f40d47669721e7a0d9fd38096f3af8f8777cb3); not a complete unit
- tour/basics.article#inline-04-L123 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 3b6395deecaeb4049ef3f95b877b75f2898a41d89d03769386385c7c3e524c2e); not a complete unit
- tour/basics.article#inline-05-L125 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 e14db1394a363626a1e8b5a13d4b8d5c3df2245765f5ee71fac52c4f451feb77); not a complete unit
- tour/basics.article#inline-06-L127 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 f87d9baee9538ab976980e0bed3a11c528b01bbda07f31afcb3e758800a5e71c); not a complete unit
- tour/basics.article#inline-07-L130 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 9f5d76947f8542802d145f3dc05eabe73161fd70acbe3b9ef93f593c75ee548a); not a complete unit
- tour/basics.article#inline-08-L132 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 e45ad6b3085c5b66f437230b861f9fea78023943badd9bd2fbd083a93ac46f2a); not a complete unit
- tour/basics.article#inline-09-L135 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 8bcf2a85dfa93de099c827a2c1a4fa16c4faec6edcf279eaeea12397f3c3bed6); not a complete unit
- tour/basics.article#inline-10-L137 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 3e99678c4ad8e3aaaf9dd3dc055f299cfa206e8b4394a4219cd08ac232f665c0); not a complete unit
- tour/basics.article#inline-11-L167 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 b00bc999950ab8b3f0b91ac7e6eb878b26bc0e013af9044b24865a4bb5230d31); not a complete unit
- tour/basics.article#inline-12-L173 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 72ffd8b6a83b3b9c1d73b725f9bc13e40c0d39709ae8e27c2629e58640a029ed); not a complete unit
- tour/basics.article#inline-13-L189 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 c29debb51ff8be567d3690819b93f1f9da32436f2f02dcbf6c671f622a9709dc); not a complete unit
- tour/basics.article#inline-14-L194 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 cfdfc30a43983de521e8a3c67f218843bd79bd4710714edae4fb5153331ccff6); not a complete unit
- tour/basics/exported-names.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 dddb6dd98a3c7188c71178c508211d9975cc00b25b6d2649e1396452ccf52460); not a complete unit
- tour/concurrency.article#inline-01-L11 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 c731b9746813a32fdafaff4d309e6023a8e8662bb8dcbc6ad49d4e024d5b69f6); not a complete unit
- tour/concurrency.article#inline-02-L15 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 33b881fe0a0b5c858befe2019b7af52993d0a7cd4094c6313fea977d74386660); not a complete unit
- tour/concurrency.article#inline-03-L27 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 23067d5b08ab23e5b132073d512c88d0cdf44061f8e9e1ef588cce56a0d8aaa3); not a complete unit
- tour/concurrency.article#inline-04-L35 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 ea027abe9c8e7bd336675e3f65b3fbebebf6783aec39986a243fcc9cdd5e5e8d); not a complete unit
- tour/concurrency.article#inline-05-L48 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 835bfa5b623fe84bc84265d0c5f06a17780e9194b7772e1c9bf74d5c6726817a); not a complete unit
- tour/concurrency.article#inline-06-L60 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 cb1ea4c6e96d46bce932d6b88f7c0087010b053ba83b633be5180670cb52fbca); not a complete unit
- tour/concurrency.article#inline-07-L86 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 0413326928564fb9464d3c307e0a1ae80a0ecfd5d643f84878cb118f0d87ae74); not a complete unit
- tour/concurrency.article#inline-08-L105 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 2df6e21b8e15af54b478f343cd395078324db23fee1c2ebaebca86cdc43e7592); not a complete unit
- tour/concurrency.article#inline-09-L124 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 ea1fcdcaaba28ff0810944507e95c474a539a7c7e3827e5c63aef34433d056a7); not a complete unit
- tour/concurrency/exercise-equivalent-binary-trees.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 184cbbd1e662b64de6e6928d4af6f2469c7bccdbc9a8b5f477c1b694b9ab02b1); not a complete unit
- tour/flowcontrol.article#inline-01-L82 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 16db8d571683f1cfe34e3d21f63923e882a80029a18229fa8b52e5cc2e057cb9); not a complete unit
- tour/flowcontrol.article#inline-02-L96 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 074e4cf3970c1256b0fdc94cd1f67172dc2901e61012b731477cda78c3374ec2); not a complete unit
- tour/flowcontrol.article#inline-03-L133 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 c6636230f3f68824f532e96e633d794a50e9be57723cbde20f181179a6872d89); not a complete unit
- tour/flowcontrol/exercise-loops-and-functions.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 65c691645723bcb2833055c22641d02e971fb57d044589144c0cabba247ba9d5); not a complete unit
- tour/methods.article#inline-01-L84 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 801ea71e71c1400e16c2464c2eb918133fdf6b11e8d6ba53b020c4b9cd607733); not a complete unit
- tour/methods.article#inline-02-L91 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 a51a1f9285706dd5c978c742d6d4b694968701d71b3db4fd28db5f3b187b9c83); not a complete unit
- tour/methods.article#inline-03-L109 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 a7b6dcb03bfb5272f2190fcf9b2d73b8f4223f77b8d3ad7e2f22f58c8b671bce); not a complete unit
- tour/methods.article#inline-04-L116 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 5eef345b42dd8ee5afee2143084902cefaff6a106e83797898e842c5e1a9309f); not a complete unit
- tour/methods.article#inline-05-L170 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 6471c0dafac5f98fb320789d7d7973311bc4429635c716e69f25f38c89e4cbab); not a complete unit
- tour/methods.article#inline-06-L205 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 b016e3043913a72757f04792a5267925b67e304d9c0573f68b63cf18111fb47e); not a complete unit
- tour/methods.article#inline-07-L222 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 081480ff1734d6e37814d128e8a3d452a53557f2cafe2e83e3706fd7333d5849); not a complete unit
- tour/methods.article#inline-08-L233 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 d14128fdef4e077c93e2c177655af0d1df1a6e09bf0d8196195293651b23bedd); not a complete unit
- tour/methods.article#inline-09-L252 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 38e69770b131ce2530a51db2370cb2177685cbf86a840efec567720aa61974c8); not a complete unit
- tour/methods.article#inline-10-L277 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 15f69abb10359dde0345db928ab7b40c0cd1332d90181bf8ba368f974ff641bd); not a complete unit
- tour/methods.article#inline-11-L301 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 118b6fb3b79499c71b59659e635414fe6a2f9f145f94d893f4fcaa5fe1c50225); not a complete unit
- tour/methods.article#inline-12-L311 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 03114c2607a61c71c90867f401e940a6c9b8f499dbc513da10b648426e7f3b24); not a complete unit
- tour/methods.article#inline-13-L330 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 4ef6f353cd3dad9e4188b2eb4c73f19d6041a9c3991ecc413b444d814d7e3b9f); not a complete unit
- tour/methods.article#inline-14-L334 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 ea4fc35919d1f956993c5e755934ab4a08faf4d88a6f96f6eeedb3ae0ebd4e7b); not a complete unit
- tour/methods.article#inline-15-L353 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 f89809eb53419662e1f21fbd5641c93b3e5bb4ffc0f01bda6796415325ed1af4); not a complete unit
- tour/methods.article#inline-16-L389 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 156060681a7539b768a81312f38f0c039e768f8d150b528be9b69d78edaffbc8); not a complete unit
- tour/methods.article#inline-17-L391 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 fc7dcb22051bc43b782be35ff43c592ad4051df252fcc33346b2c996d5f4c6bb); not a complete unit
- tour/methods/exercise-images.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 774bbc5fa4b7284d4302f5a504f34642b89d276f845b455ecbca5abd4cada25f); not a complete unit
- tour/methods/exercise-reader.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 7f5acb25434e41b1641fffd602725903cd2e217fb7f8809596c4369ac0900137); not a complete unit
- tour/methods/exercise-rot-reader.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 9972063506e3a9193e7761d1118c2bd3680ed8a56f855f788403b8a193edbaa2); not a complete unit
- tour/methods/interfaces.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 2530ef039212284377a46fe1063db6206d8db3a42a8138c6eed44275bb7eb10b); not a complete unit
- tour/moretypes.article#inline-01-L14 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 c21ab30d0bc7342a5d95f9b419c895f0d81a112ea3a41382fa87fa18d8551aff); not a complete unit
- tour/moretypes.article#inline-02-L18 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 e63327021e32acb558ffc0f76380c62dfddf3c7d704358c33bcbd45da5a90d08); not a complete unit
- tour/moretypes.article#inline-03-L23 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 44cea08a38de8c697a18e0c1a38a63f98591a244f3de22bcb9a05a9e5ef5d056); not a complete unit
- tour/moretypes.article#inline-04-L72 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 93ed85163b212c5bccc774b28b9c0055f60cf7e42193904bfc4b1412088e8314); not a complete unit
- tour/moretypes.article#inline-05-L95 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 00ac337247b75420ffc2e6b287df2792f9e1f4f550deea0c3414c049d1ac78c8); not a complete unit
- tour/moretypes.article#inline-06-L103 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 6c6f5c14fa5af5e17105a6d0d485bff690305a558b10dd3a4d7cc11d76edfe89); not a complete unit
- tour/moretypes.article#inline-07-L127 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 4160fe2891df5ff3050434fd5016cbc6bf6a826974ce1242cdc1adc0b80ada62); not a complete unit
- tour/moretypes.article#inline-08-L132 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 38fe1c93e532337d3e11504cbde8c53db47a353608e6cc2833c2e9f5ad72d343); not a complete unit
- tour/moretypes.article#inline-09-L145 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 93ed85163b212c5bccc774b28b9c0055f60cf7e42193904bfc4b1412088e8314); not a complete unit
- tour/moretypes.article#inline-10-L149 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 f167402cdc88fbad72d720e4584bd0e34a84e32768a3965fa157d148f894574b); not a complete unit
- tour/moretypes.article#inline-11-L195 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 e86ddfeb2590c463cde7e78d4afd2cbb8cff325cc870d3750fc2d5205151e92f); not a complete unit
- tour/moretypes.article#inline-12-L199 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 7338c5b5aa18b91860cea3c6bace6487e916ae02e69b8ed7217563bea8cb90d9); not a complete unit
- tour/moretypes.article#inline-13-L201 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 2309dec82c10b31d8f3436e29e179352743fbe0b258cf2da7a6708ad0d076fe2); not a complete unit
- tour/moretypes.article#inline-14-L220 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 e0fae10cc12e1a46d8a674c537ee8ff7b0daaee3545186407f8a861e0a6ac512); not a complete unit
- tour/moretypes.article#inline-15-L299 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 98510a98fbbbd01eca3ce5503106134fecb035d36ac6f37a3004ce4dc6439600); not a complete unit
- tour/moretypes.article#inline-16-L303 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 6eda3a4dea8c4948250ba623a405798322fb313286016aa846cbae60a9d136ea); not a complete unit
- tour/moretypes.article#inline-17-L307 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 fab75903ad097b2712e884086a7ca957a0701c811725de2e01f838dd9dff0312); not a complete unit
- tour/moretypes.article#inline-18-L311 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 0e4bea0695ec2b5e07401c4bc70b5b30e7d67ffe62c0d1fb6d70b3a6f5a197bc); not a complete unit
- tour/moretypes.article#inline-19-L319 (not-applicable): tour article_inline_block excluded_fragment: bytes not committed (inventory pins upstream sha256 303464d68c81c2e2333d3210af4fc0369ce647f9cb1087ca94690f197bb1aa7b); not a complete unit
- tour/moretypes/exercise-fibonacci-closure.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 a04ba02819628af3c95c045330a5ce733f9f182e755e6ba74e78c0aea602530c); not a complete unit
- tour/moretypes/exercise-slices.go (not-applicable): tour lesson_play_program excluded_fragment: bytes not committed (inventory pins upstream sha256 62284c50db0f192347f6a25f704d8d5781a3e8d838644ba25db5de953ce14cc9); not a complete unit
- pin/tour (not-applicable): provenance pin golang.org/x/website@v0.0.0-20260903033311-c4a9d59f9775 (c4a9d59f9775d994f1700d18fa37414c3c85fa7b), checked by tools/tour/validate.sh
- gbe/UPSTREAM-README.md (not-applicable): go-by-example provenance: not a program (classification.tsv not_a_program)
- gbe/embed-directive/folder/file1.hash (not-applicable): go-by-example runtime_asset: not a program (classification.tsv not_a_program)
- gbe/embed-directive/folder/file2.hash (not-applicable): go-by-example runtime_asset: not a program (classification.tsv not_a_program)
- gbe/embed-directive/folder/single_file.txt (not-applicable): go-by-example runtime_asset: not a program (classification.tsv not_a_program)
- pin/go-by-example (not-applicable): provenance pin https://github.com/mmcgrana/gobyexample.git@7d705626375ba0263b616865a286e1587d6989c8; classification.tsv and inventory.tsv are joined by tools/go-by-example/validate.sh
- bpp/00_superset_posix2016/01_posix_baseline.sh (planned): tests/manifest.tsv planned (posix-baseline P0); status declared there, not reclassified
- bpp/01_syntax/01_decl_assign.bpp (planned): tests/manifest.tsv planned (declarations P1); status declared there, not reclassified
- bpp/03_concurrency/01_chan_select.bpp (planned): tests/manifest.tsv planned (concurrency P4); status declared there, not reclassified
- bpp/06_go126_features/01_clear_builtin.bpp (planned): tests/manifest.tsv planned (builtins G127); status declared there, not reclassified
- s6/M09 (planned): S6 gap, refused by sh/syntax today (`}` can only be used to close a block); failure phase measured at the syntax seam in checkpoint 2; a gap is not an exclusion
- s6/M10 (planned): S6 gap, refused by sh/syntax today (for condition must be a scalar expression); failure phase measured at the syntax seam in checkpoint 2; a gap is not an exclusion
- s6/M11 (planned): S6 gap, refused by sh/syntax today (syntax error near unexpected token `float64]'); failure phase measured at the syntax seam in checkpoint 2; a gap is not an exclusion
- s6-support/I03-main.go (not-applicable): embed target of s6/I03; parsed as that row's Go payload
- s6-support/I07.nomod (not-applicable): empty marker: s6/I07 runs without a go.mod
- delta/accept/calc.py (not-applicable): embed target file, not a script
