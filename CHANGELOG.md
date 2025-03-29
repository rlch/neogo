# Changelog

## [1.0.6](https://github.com/rlch/neogo/compare/v1.0.5...v1.0.6) (2025-03-28)


### Bug Fixes

* Allow unmarshalling to slices of size 1 ([358aedf](https://github.com/rlch/neogo/commit/358aedf607e57560411c8dcfe2cb865c2412e85f))

## [1.0.5](https://github.com/rlch/neogo/compare/v1.0.4...v1.0.5) (2025-03-28)


### Bug Fixes

* Remove another print ([8f6b3e5](https://github.com/rlch/neogo/commit/8f6b3e54c298d2f8b606dc6b95cdc275ce0bee86))
* Remove print ([328c2c7](https://github.com/rlch/neogo/commit/328c2c75f8c2622732c4d0c8d132430a4d06d933))

## [1.0.4](https://github.com/rlch/neogo/compare/v1.0.3...v1.0.4) (2025-03-28)


### Bug Fixes

* Avoid downcasting abstract types upon registration ([c241f47](https://github.com/rlch/neogo/commit/c241f471036b1b9ab06528b2973db18c5a3aac05))
* Handle unmarshalling to empty slice and slice of length 1 ([894de73](https://github.com/rlch/neogo/commit/894de73fbfe290d69b917968d5c7602089054d99))

## [1.0.3](https://github.com/rlch/neogo/compare/v1.0.2...v1.0.3) (2025-02-12)


### Bug Fixes

* Handle nil bookmarks ([52456f7](https://github.com/rlch/neogo/commit/52456f769cea5c4f59be4ecfd90e3083a65dc583))

## [1.0.2](https://github.com/rlch/neogo/compare/v1.0.1...v1.0.2) (2025-01-28)


### Bug Fixes

* Add Label stub type ([51e856b](https://github.com/rlch/neogo/commit/51e856bbcde418d5f439c3e7f3195e8dd1409fe1))
* Only require concrete node labels for abstract deserialization ([5cf6104](https://github.com/rlch/neogo/commit/5cf61043e37db9017ff51e44fad7ce092c1e9fd5))

## [1.0.1](https://github.com/rlch/neogo/compare/v1.0.0...v1.0.1) (2024-09-26)


### Bug Fixes

* Ensure driver is propagated to session ([ef1f2ee](https://github.com/rlch/neogo/commit/ef1f2ee9bbe42c720100fcc0cac2a87c2c0187b0))

## 1.0.0 (2024-08-08)


### Features

* Add explicit transactions; API cleanups ([#12](https://github.com/rlch/neogo/issues/12)) ([0875fc4](https://github.com/rlch/neogo/commit/0875fc4927421a7d634e019b7359f268962a2e59))
* Add mock Neo4J client ([#9](https://github.com/rlch/neogo/issues/9)) ([e57f2aa](https://github.com/rlch/neogo/commit/e57f2aa0ded8cac866c18396e22e0bb773582490))
* Add Print to client ([af31477](https://github.com/rlch/neogo/commit/af31477be815d331cf6b3af6c2b2ae83398a8311))
* Add PropsExpr API ([ead28ff](https://github.com/rlch/neogo/commit/ead28ff36e5d6e72d4cbb98c578239fa334ee783))
* Add stream with params ([#8](https://github.com/rlch/neogo/issues/8)) ([58e7457](https://github.com/rlch/neogo/commit/58e74574f580f78d5e37764029c82898a9cce111))
* Added support for injecting params in the query ([#7](https://github.com/rlch/neogo/issues/7)) ([e7dc293](https://github.com/rlch/neogo/commit/e7dc293b33808249e6c77b262161a637e4a32963))
* Allow props to be injected in MATCH and MERGE clauses ([37bd5ab](https://github.com/rlch/neogo/commit/37bd5ab6933dad06b55287ca7ea5139af49e2085))
* Expose ExtractNodeLabels/RelationshipType ([f39660f](https://github.com/rlch/neogo/commit/f39660f5c15d511331612348565e24300ab4bf02))
* Implement Stream API ([#4](https://github.com/rlch/neogo/issues/4)) ([42836f1](https://github.com/rlch/neogo/commit/42836f1931f422cd685b406b9cfda8a3d712b97f))
* Init commit ([4adb744](https://github.com/rlch/neogo/commit/4adb7447c3183906fa3e4ecc132d2bd74038bfde))
* Support abstract node collections ([70f5a16](https://github.com/rlch/neogo/commit/70f5a16b8e275c6689100667db26eba34bf9113d))
* Support causal consistency ([3c85566](https://github.com/rlch/neogo/commit/3c85566e4fb72ee56075b1071cb048a8bccd23da))


### Bug Fixes

* **actions:** Skip long tests ([ac1b1b8](https://github.com/rlch/neogo/commit/ac1b1b8a8a90580ee12c4ed3faedb1d87928f712))
* Add check for non-nil parameters with expression and aliasing ([#6](https://github.com/rlch/neogo/issues/6)) ([c3415cf](https://github.com/rlch/neogo/commit/c3415cfe546c6eb2d80555b091782f6eb0b81173))
* Add constraint to abstract nodes being interfaces ([fb143d9](https://github.com/rlch/neogo/commit/fb143d9529b684d2318a39ff2410fecd26819492))
* Add joinedErrors API ([655fd41](https://github.com/rlch/neogo/commit/655fd41c7a9e1ed9d8a55cbc7a20b5bc7723aceb))
* Add multi-polymorphism examples ([9e0246d](https://github.com/rlch/neogo/commit/9e0246de80de38b480c6bad226c2cd6967967df8))
* Add nil-check to recordType ([b8b5a25](https://github.com/rlch/neogo/commit/b8b5a2507a48b680943b71d35f7ff1a24449b3e5))
* Allow binding [][]AbstractImpl ([066f308](https://github.com/rlch/neogo/commit/066f30824cd56bafaeeb95eb4f1ca2b61b933545))
* Avoid resetting builder in Print ([270c565](https://github.com/rlch/neogo/commit/270c56510cd038367186bb7c4acc06f05eddbeb3))
* Defer canBindSubtype error ([8b90e2c](https://github.com/rlch/neogo/commit/8b90e2c8b6d72589e0408dec2f07cb50e9c1b726))
* Ensure ctx is finished ([886465d](https://github.com/rlch/neogo/commit/886465d4aa3018669b865f0833ab7367520e0a0a))
* Ensure docker tests are all short ([88c801e](https://github.com/rlch/neogo/commit/88c801e15e62a3e94c60a24bbad4c393bbfe256c))
* Ensure isWrite is true when using Cypher() ([d69d8ae](https://github.com/rlch/neogo/commit/d69d8ae4ce4b47ec4ca043e84526ed8b4e3d1cfc))
* Ensure recursion is invoked ([71ecb3d](https://github.com/rlch/neogo/commit/71ecb3d828b4e6bdc23e16a11fbc3dbb810f6b97))
* Expose Config ([67c13bc](https://github.com/rlch/neogo/commit/67c13bc21c630ecdd1fa23279dc40f2a50ae98fa))
* Expose Runner ([bd5cfc6](https://github.com/rlch/neogo/commit/bd5cfc6d66cc0ab7fe8bf0420b75e8f9087bbd2e))
* Expose TxWork ([71d3883](https://github.com/rlch/neogo/commit/71d38837fd98a7677943dd050ddfef0c94e49f69))
* Handle nested structs ([#11](https://github.com/rlch/neogo/issues/11)) ([8cc6249](https://github.com/rlch/neogo/commit/8cc62498298dfa1676c70d49c4f60f6429ed7d28))
* Handle nil case ([1d4c2af](https://github.com/rlch/neogo/commit/1d4c2afa9fe705d2494039ccd44f1225175bb7f7))
* Improve decision framework for single/multi unmarshalling ([32639dd](https://github.com/rlch/neogo/commit/32639ddd3356376655334137ea35abe02dda49ec))
* Improve layout ([c861ad6](https://github.com/rlch/neogo/commit/c861ad694c997bfa9a627e7c84c9394f31c5ce8d))
* Improve nil handling in deserialization ([5b486e7](https://github.com/rlch/neogo/commit/5b486e7d73fa86b37296c683c9f818ec062b25b8))
* Initialize Querier inside Yielder ([#10](https://github.com/rlch/neogo/issues/10)) ([a37644b](https://github.com/rlch/neogo/commit/a37644b7737a534b712956366c44f6258ab14c6e))
* Linter errors ([cc14649](https://github.com/rlch/neogo/commit/cc1464933a16024f048d30394a196a9fb10726af))
* Linter errors; ensure tests use correct instance of *testing.T ([eb984b9](https://github.com/rlch/neogo/commit/eb984b92e7b44f263b25e29938eb744287680cab))
* Only extract labels from anonymous structs ([64d5f87](https://github.com/rlch/neogo/commit/64d5f871129b95b6196e6db326ab9e0fe4abaa4e))
* Only extract neo4j tags from struct type fields ([#5](https://github.com/rlch/neogo/issues/5)) ([399bb27](https://github.com/rlch/neogo/commit/399bb27a4fb67a9ab36cfb2b83cb87caaedbebfc))
* Pointer-recurse into outer type before extracting tags ([1d4d899](https://github.com/rlch/neogo/commit/1d4d89960e8621a3ea912d86fc5d93e30a0ca01e))
* Prioritize Valuer over recursion ([09ccea2](https://github.com/rlch/neogo/commit/09ccea24d86f2b591ea115425370a77101b260fb))
* Recurse inside entity when computing props ([e13ad93](https://github.com/rlch/neogo/commit/e13ad937c39b173f527b30c936a1ff6dc0505cf4))
* Recurse into pointers to abstract ndoes ([ba54418](https://github.com/rlch/neogo/commit/ba54418e7d4e689eec53ea902cd761b241de6083))
* Relax constraint on unmarshaling param values ([7fb96ad](https://github.com/rlch/neogo/commit/7fb96adad7010ac13cf4f3cb99cab3fa5d6eaf4d))
* Remove Select API ([366bf97](https://github.com/rlch/neogo/commit/366bf97309221d5382a9dc76c1152415f6c53e79))
* Remove unused function ([0729f0a](https://github.com/rlch/neogo/commit/0729f0a9044d829d45bf51c11d03b8c4464346e6))
* Rename MergeOptions to Merge ([4140139](https://github.com/rlch/neogo/commit/41401392a4f2444443f6d7ef1c1f1d1ba24524a2))
* Set isWrite properly in expressions and Cypher ([6ae4b91](https://github.com/rlch/neogo/commit/6ae4b9106ff10c269f74d5c8b77a557980ed9b7e))
* Support binding comopsite fields ([aa4977d](https://github.com/rlch/neogo/commit/aa4977d7baf32cca48c9425c7f87ba33993ae16f))
* Support multi-inheritance; allow concrete implementers to also be registered ([a979f59](https://github.com/rlch/neogo/commit/a979f59a8731b8649dec7b493dfe2bf6646ad6e0))
* Support nil+registered / non-nil+base-type instantiation of Abstract nodes ([43d742a](https://github.com/rlch/neogo/commit/43d742ac699ab9d8dcc3d1e29d9c87b1c7093b1c))
* Test deepsource ([c27f6bd](https://github.com/rlch/neogo/commit/c27f6bdb615011bb51d667f01fa3f89958b5402e))
