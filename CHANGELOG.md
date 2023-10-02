# Changelog

## 1.0.0 (2023-10-02)


### Features

* Add mock Neo4J client ([#9](https://github.com/rlch/neogo/issues/9)) ([e57f2aa](https://github.com/rlch/neogo/commit/e57f2aa0ded8cac866c18396e22e0bb773582490))
* Add Print to client ([af31477](https://github.com/rlch/neogo/commit/af31477be815d331cf6b3af6c2b2ae83398a8311))
* Add PropsExpr API ([ead28ff](https://github.com/rlch/neogo/commit/ead28ff36e5d6e72d4cbb98c578239fa334ee783))
* Add stream with params ([#8](https://github.com/rlch/neogo/issues/8)) ([58e7457](https://github.com/rlch/neogo/commit/58e74574f580f78d5e37764029c82898a9cce111))
* Added support for injecting params in the query ([#7](https://github.com/rlch/neogo/issues/7)) ([e7dc293](https://github.com/rlch/neogo/commit/e7dc293b33808249e6c77b262161a637e4a32963))
* Allow props to be injected in MATCH and MERGE clauses ([37bd5ab](https://github.com/rlch/neogo/commit/37bd5ab6933dad06b55287ca7ea5139af49e2085))
* Implement Stream API ([#4](https://github.com/rlch/neogo/issues/4)) ([42836f1](https://github.com/rlch/neogo/commit/42836f1931f422cd685b406b9cfda8a3d712b97f))
* Init commit ([4adb744](https://github.com/rlch/neogo/commit/4adb7447c3183906fa3e4ecc132d2bd74038bfde))


### Bug Fixes

* **actions:** Skip long tests ([ac1b1b8](https://github.com/rlch/neogo/commit/ac1b1b8a8a90580ee12c4ed3faedb1d87928f712))
* Add check for non-nil parameters with expression and aliasing ([#6](https://github.com/rlch/neogo/issues/6)) ([c3415cf](https://github.com/rlch/neogo/commit/c3415cfe546c6eb2d80555b091782f6eb0b81173))
* Add constraint to abstract nodes being interfaces ([fb143d9](https://github.com/rlch/neogo/commit/fb143d9529b684d2318a39ff2410fecd26819492))
* Avoid resetting builder in Print ([270c565](https://github.com/rlch/neogo/commit/270c56510cd038367186bb7c4acc06f05eddbeb3))
* Defer canBindSubtype error ([8b90e2c](https://github.com/rlch/neogo/commit/8b90e2c8b6d72589e0408dec2f07cb50e9c1b726))
* Ensure docker tests are all short ([88c801e](https://github.com/rlch/neogo/commit/88c801e15e62a3e94c60a24bbad4c393bbfe256c))
* Ensure isWrite is true when using Cypher() ([d69d8ae](https://github.com/rlch/neogo/commit/d69d8ae4ce4b47ec4ca043e84526ed8b4e3d1cfc))
* Ensure recursion is invoked ([71ecb3d](https://github.com/rlch/neogo/commit/71ecb3d828b4e6bdc23e16a11fbc3dbb810f6b97))
* Expose Config ([67c13bc](https://github.com/rlch/neogo/commit/67c13bc21c630ecdd1fa23279dc40f2a50ae98fa))
* Expose TxWork ([71d3883](https://github.com/rlch/neogo/commit/71d38837fd98a7677943dd050ddfef0c94e49f69))
* Improve decision framework for single/multi unmarshalling ([32639dd](https://github.com/rlch/neogo/commit/32639ddd3356376655334137ea35abe02dda49ec))
* Initialize Querier inside Yielder ([#10](https://github.com/rlch/neogo/issues/10)) ([a37644b](https://github.com/rlch/neogo/commit/a37644b7737a534b712956366c44f6258ab14c6e))
* Linter errors ([cc14649](https://github.com/rlch/neogo/commit/cc1464933a16024f048d30394a196a9fb10726af))
* Linter errors; ensure tests use correct instance of *testing.T ([eb984b9](https://github.com/rlch/neogo/commit/eb984b92e7b44f263b25e29938eb744287680cab))
* Only extract neo4j tags from struct type fields ([#5](https://github.com/rlch/neogo/issues/5)) ([399bb27](https://github.com/rlch/neogo/commit/399bb27a4fb67a9ab36cfb2b83cb87caaedbebfc))
* Pointer-recurse into outer type before extracting tags ([1d4d899](https://github.com/rlch/neogo/commit/1d4d89960e8621a3ea912d86fc5d93e30a0ca01e))
* Recurse inside entity when computing props ([e13ad93](https://github.com/rlch/neogo/commit/e13ad937c39b173f527b30c936a1ff6dc0505cf4))
* Recurse into pointers to abstract ndoes ([ba54418](https://github.com/rlch/neogo/commit/ba54418e7d4e689eec53ea902cd761b241de6083))
* Relax constraint on unmarshaling param values ([7fb96ad](https://github.com/rlch/neogo/commit/7fb96adad7010ac13cf4f3cb99cab3fa5d6eaf4d))
* Remove Select API ([366bf97](https://github.com/rlch/neogo/commit/366bf97309221d5382a9dc76c1152415f6c53e79))
* Rename MergeOptions to Merge ([4140139](https://github.com/rlch/neogo/commit/41401392a4f2444443f6d7ef1c1f1d1ba24524a2))
* Support binding comopsite fields ([aa4977d](https://github.com/rlch/neogo/commit/aa4977d7baf32cca48c9425c7f87ba33993ae16f))
* Support multi-inheritance; allow concrete implementers to also be registered ([a979f59](https://github.com/rlch/neogo/commit/a979f59a8731b8649dec7b493dfe2bf6646ad6e0))
* Support nil+registered / non-nil+base-type instantiation of Abstract nodes ([43d742a](https://github.com/rlch/neogo/commit/43d742ac699ab9d8dcc3d1e29d9c87b1c7093b1c))
