# Changelog

All notable changes to ServicePro will be documented in this file.

## [0.4.0] - 2026-01-28

### Release Features

| Type             | Description                                                                          | Commit                                                                                                 |
| ---------------- | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| ✨ Features      | Created full backend API to handle all payment systems                               | [`1919e3b`](https://github.com/javaknight1/servicepro/commit/1919e3bb0fb63b16cfd39e8d6efe0dba37c5caf1) |
| ✨ Features      | Integrated backend payment APIs to UI                                                | [`c4cc7a5`](https://github.com/javaknight1/servicepro/commit/c4cc7a57485d2534cd0063f3c886a3dd41c9b581) |
| ✨ Features      | Added backend API to send invites to users registered and not registered to platform | [`cc14479`](https://github.com/javaknight1/servicepro/commit/cc14479282ca92443b8aa1444a5009d4a5cc1a20) |
| ✨ Features      | Integrated new invite API into frontend                                              | [`af2e618`](https://github.com/javaknight1/servicepro/commit/af2e6185eb4156527c9ca1f163dc86598020d5eb) |
| 🎯 Minor Changes | Added rating limiting to API                                                         | [`8380da5`](https://github.com/javaknight1/servicepro/commit/8380da561214805ae17379b09f6ae1f02f27991f) |
| 🐛 Bug Fixes     | Fixed potential sql injection                                                        | [`1c55eb1`](https://github.com/javaknight1/servicepro/commit/1c55eb1b4ad792aab5b9589b25078f0daede6d8f) |
| 🐛 Bug Fixes     | Added some security HTTP validation                                                  | [`4a04a96`](https://github.com/javaknight1/servicepro/commit/4a04a96164dba827e7ac6a6970b7779e028cee43) |
| 🐛 Bug Fixes     | Integrated CORS middleware                                                           | [`66b040f`](https://github.com/javaknight1/servicepro/commit/66b040f1573cd71fc3117d13e51e226e77585359) |
| 🐛 Bug Fixes     | Fixed incorrect path to cmd/main.go                                                  | [`bdc6b71`](https://github.com/javaknight1/servicepro/commit/bdc6b71d41168e6919b5cc621e539f414a8dd03e) |
| 🐛 Bug Fixes     | Fixed issue gotten when editing invoices                                             | [`0b00ef0`](https://github.com/javaknight1/servicepro/commit/0b00ef0082248e20363d55bf37868d5f6515f56e) |

### Dev Features

| Type             | Description                                                            | Commit                                                                                                 |
| ---------------- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ♻️ Refactoring   | Updated comments and references to signal s3 agnostic                  | [`1b99ceb`](https://github.com/javaknight1/servicepro/commit/1b99ceb9fa54ed678b48813a6e73198fe36110ae) |
| ♻️ Refactoring   | Clean up our config for the backend                                    | [`77fa078`](https://github.com/javaknight1/servicepro/commit/77fa078a097906fd10d603aedcbf50786b91c940) |
| ♻️ Refactoring   | Simplify our routing library                                           | [`e435415`](https://github.com/javaknight1/servicepro/commit/e4354159dda097653f535f6cb1620f1e24396fdb) |
| ✅ Tests         | Added SMTP local service to docker compose for local email development | [`47d0a70`](https://github.com/javaknight1/servicepro/commit/47d0a709670f03acf9efc6290864dc620dc0a025) |
| ✅ Tests         | Fixed some backend unit tests for invoice                              | [`a7dc531`](https://github.com/javaknight1/servicepro/commit/a7dc531fb37f84d3571d1964f1c84c3d12f62ddd) |
| 📚 Documentation | Update our docs with new info on env vars                              | [`25e0f9e`](https://github.com/javaknight1/servicepro/commit/25e0f9e61a9fcc3eafad36ae4644f8aa11d2a367) |
| 🔧 Miscellaneous | Migrated .sql migrations into single migration file                    | [`dd89dae`](https://github.com/javaknight1/servicepro/commit/dd89dae55bf729f194e60bd69ef47b91b6f6491f) |
| 🚀 CI/CD         | Fixed a linting error with main.go                                     | [`0a43808`](https://github.com/javaknight1/servicepro/commit/0a43808c6a4e67ad7526474d8b935d613b4ad566) |

## [0.3.0] - 2026-01-24

### Release Features

| Type             | Description                                                                | Commit                                                                                                 |
| ---------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ✨ Features      | Add endpoints to allow users to update membership status for organizations | [`9db1c7a`](https://github.com/javaknight1/servicepro/commit/9db1c7ae23cf0bab4628ca9cf1628d64a17a5aec) |
| 🎯 Minor Changes | Added new endpoint to give users a first name and last name                | [`7346cdd`](https://github.com/javaknight1/servicepro/commit/7346cdd4f77f50881b56c253983b75c0aca9ea2b) |
| 🎯 Minor Changes | Get latest first and last name and edit in Profile                         | [`d677665`](https://github.com/javaknight1/servicepro/commit/d67766541218bd094daa41d0955a45aa7bebc7ae) |
| 🎯 Minor Changes | Updated the UI to include membership features                              | [`c1fd85d`](https://github.com/javaknight1/servicepro/commit/c1fd85df33069e227fe786eab05d6b0750118f7a) |
| 🐛 Bug Fixes     | Fixed some warnings getting emitted by commits                             | [`e0d5280`](https://github.com/javaknight1/servicepro/commit/e0d52805f670a0ea260842c6acb8afa7bc10905d) |
| 🐛 Bug Fixes     | Fixed issue with prettier versions incompatible                            | [`a741b7c`](https://github.com/javaknight1/servicepro/commit/a741b7c552a88654590d5f80823fa60cd2e1e799) |
| 🐛 Bug Fixes     | Resolved eslint warnings in UI                                             | [`768475c`](https://github.com/javaknight1/servicepro/commit/768475cba7b967af10cec37cfb5bca5285323132) |
| 🐛 Bug Fixes     | Resolve wanrings that came from updating eslint                            | [`52d58f1`](https://github.com/javaknight1/servicepro/commit/52d58f1603bfc0b72db6e8a15d35b118423a9134) |

### Dev Features

| Type             | Description                                            | Commit                                                                                                 |
| ---------------- | ------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| ✅ Tests         | Fixed unuit tests after eslint changes                 | [`21faf03`](https://github.com/javaknight1/servicepro/commit/21faf03ef01765d600192e30fd5e7c93a61eaeb6) |
| 🔧 Miscellaneous | Auto generate prettier and lint                        | [`d0108aa`](https://github.com/javaknight1/servicepro/commit/d0108aa6065b93b68eddd2060ae8f1ef0b76378a) |
| 🔧 Miscellaneous | Remove workflow that we aren't using right now         | [`638df07`](https://github.com/javaknight1/servicepro/commit/638df07ef3afe4d80b1a2b1e867b5f2c9523d8fe) |
| 🔧 Miscellaneous | Created CLAUDE.md to help with development             | [`ef0fdab`](https://github.com/javaknight1/servicepro/commit/ef0fdab1cfb2eb52a3a0c4cfd257bedb31e81f67) |
| 🔧 Miscellaneous | Include Minio in makefile commands                     | [`a7376ce`](https://github.com/javaknight1/servicepro/commit/a7376ce556042e159f423a8618943df720b09a9a) |
| 🔧 Miscellaneous | Updated @typescript-eslint/typescript-estree to 8.53.1 | [`786024e`](https://github.com/javaknight1/servicepro/commit/786024e4946dd6fe67fc7db36bdebe297ad4e9a5) |
| 🔧 Miscellaneous | Disable some lintings:                                 | [`780fec6`](https://github.com/javaknight1/servicepro/commit/780fec6bbf935ae670bb8585a38c7d366bcf56d4) |
| 🔧 Miscellaneous | Require unit tests on pre-commit                       | [`1dc6bee`](https://github.com/javaknight1/servicepro/commit/1dc6bee4cfa6d22c963f10e2b9517c29fb7340ec) |

## [0.2.0] - 2026-01-24

### Release Features

| Type             | Description                                                    | Commit                                                                                                 |
| ---------------- | -------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ✨ Features      | Created endpoint to CRUD a user's profile image                | [`b157f50`](https://github.com/javaknight1/servicepro/commit/b157f507c1ee116cc61ba7dd6107ba2a0072d354) |
| ✨ Features      | Created frontend components to create profile pic for users    | [`2066c0f`](https://github.com/javaknight1/servicepro/commit/2066c0f9fcbb6861ea0e1c602de58002bc2732de) |
| 🎯 Minor Changes | Created new version endpoint to return current running version | [`e1965d5`](https://github.com/javaknight1/servicepro/commit/e1965d5e95fc7bfe0ea02e1479f9b3c770b07e0b) |

### Dev Features

| Type             | Description                                                           | Commit                                                                                                 |
| ---------------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ♻️ Refactoring   | Removed unused gitops                                                 | [`b681a1f`](https://github.com/javaknight1/servicepro/commit/b681a1ff61a331463589ce2c31cdd41f42f96567) |
| ♻️ Refactoring   | Centralized config and migrated any client services to factory design | [`e7d827e`](https://github.com/javaknight1/servicepro/commit/e7d827e886cee416863bd1856f7176729870da6a) |
| ♻️ Refactoring   | Removed code not being used in backend and recorded into TODO         | [`f3ea4d8`](https://github.com/javaknight1/servicepro/commit/f3ea4d82524852b70f1fcd2e10855885efd2ba75) |
| ♻️ Refactoring   | Supporting only S3 compatible storage options                         | [`bc992f6`](https://github.com/javaknight1/servicepro/commit/bc992f67a48b00eb17371f06a87862d4e5744b8e) |
| ✅ Tests         | Fixed docker compose and Makefile to work correctly                   | [`6051f9f`](https://github.com/javaknight1/servicepro/commit/6051f9f863988008215264b51796a9bc5448ae8c) |
| ✅ Tests         | Fixed many of the broken unit tests                                   | [`8e1ba2b`](https://github.com/javaknight1/servicepro/commit/8e1ba2b8a81a0aa0b7d3cb4f2402711a7271c2b3) |
| ✅ Tests         | Fixed some unit tests                                                 | [`b7d56ad`](https://github.com/javaknight1/servicepro/commit/b7d56ad39077e86b98936dbdf4b24177e1c2f7dd) |
| ✅ Tests         | Fixed all unit tests for frontend                                     | [`9f25cd8`](https://github.com/javaknight1/servicepro/commit/9f25cd86e420ee7101908e478d1a20025ca94034) |
| ✅ Tests         | Fixed breaking unit tests for backend and frontend                    | [`a1841c7`](https://github.com/javaknight1/servicepro/commit/a1841c717032f21a69ce64e7f10efc26d4dc3b58) |
| ✅ Tests         | Streamlined a lot of our scripts                                      | [`a4d03a5`](https://github.com/javaknight1/servicepro/commit/a4d03a56367f42748441fe0665855f8663a3dd94) |
| ✅ Tests         | Fixed frontend not being able to run                                  | [`488e1d1`](https://github.com/javaknight1/servicepro/commit/488e1d177e9d622cfddf0cc445cf440c444b83e3) |
| ✅ Tests         | Removed test docker compose                                           | [`f91ee65`](https://github.com/javaknight1/servicepro/commit/f91ee652d9ec55d53f40a604ee7603135bb5d958) |
| ✅ Tests         | Fixed unit tests with new profile pic handler                         | [`93ca0bc`](https://github.com/javaknight1/servicepro/commit/93ca0bca7e99b4049cebf43a871c1383359bcb8f) |
| 📚 Documentation | Streamlined and cleaned up all the docs                               | [`ba6de95`](https://github.com/javaknight1/servicepro/commit/ba6de95028a9207439304b1628fb31d9f3ca7336) |
| 🔧 Miscellaneous | Updated .gitignore for removing terraform, helm, and gitops           | [`9240607`](https://github.com/javaknight1/servicepro/commit/9240607051d8022df10cf32bf722685e96d74224) |
| 🔧 Miscellaneous | Removed useless commands in Makefile                                  | [`b766aec`](https://github.com/javaknight1/servicepro/commit/b766aeca5834f7acdd5c5229d4999faf1cbd6564) |
| 🔧 Miscellaneous | Cleaned up change log formatting                                      | [`9a0aff4`](https://github.com/javaknight1/servicepro/commit/9a0aff4c79b5f4adc4dbf41acd24e6fd0f78f2bb) |
| 🚀 CI/CD         | Update linting job                                                    | [`5997ffe`](https://github.com/javaknight1/servicepro/commit/5997ffe19c81b11e78e075123cb16862e477eea3) |
| 🚀 CI/CD         | Ignore linting for some files                                         | [`152a536`](https://github.com/javaknight1/servicepro/commit/152a536c473063df5accf0e2d48150bc71f586e8) |
| 🚀 CI/CD         | Streamline the Makefile                                               | [`e70e4fa`](https://github.com/javaknight1/servicepro/commit/e70e4fac1098d8cca53f7489ffa36a3a7e42851c) |
| 🚀 CI/CD         | Fixed some linting issues from our CI job                             | [`d021036`](https://github.com/javaknight1/servicepro/commit/d0210360d42862ec3678fd8a81907128123df95c) |
| 🚀 CI/CD         | Clean up all our linting                                              | [`8f9aaed`](https://github.com/javaknight1/servicepro/commit/8f9aaed32e7259ff5c56f972d304259d5c54eae6) |
| 🚀 CI/CD         | Cleaned up all our ci jobs                                            | [`f5f02d1`](https://github.com/javaknight1/servicepro/commit/f5f02d1eccf9202173eada36e43006aa98156df7) |
| 🚀 CI/CD         | Fixed failing remote unit tests for frontend                          | [`d6fac12`](https://github.com/javaknight1/servicepro/commit/d6fac126450305edc39b2092ebe5544a1d57b005) |
| 🚀 CI/CD         | Pretty linting files                                                  | [`9de1414`](https://github.com/javaknight1/servicepro/commit/9de14142670af34428ed816ad464d8b7d80bf206) |
| 🚀 CI/CD         | Fixed linting errors                                                  | [`3a6489f`](https://github.com/javaknight1/servicepro/commit/3a6489fec8f922193798e73e9b532dee32667371) |
| 🚀 CI/CD         | Updated output of change log for new releases                         | [`f189672`](https://github.com/javaknight1/servicepro/commit/f189672de35e80e6d11f9425853b50ed8a9c6b79) |
| 🚀 CI/CD         | Output major and minor commits                                        | [`f2da4fc`](https://github.com/javaknight1/servicepro/commit/f2da4fcd43acdd37221b1af8d780fc8af08cfe65) |
| 🚀 CI/CD         | Fixed change log generation commit link                               | [`7d2b84d`](https://github.com/javaknight1/servicepro/commit/7d2b84d9090abc2f2a542c92e62e89bc35dde15c) |
| 🚀 CI/CD         | Fixed change log white space generated                                | [`671c64e`](https://github.com/javaknight1/servicepro/commit/671c64e89f4dc594df2afa1f5b63374bc9cd51ed) |

## [0.1.0] - 2026-01-23

### Dev Features

| Type             | Description                                                               | Commit                                                                                                 |
| ---------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| 🔧 Miscellaneous | Removed terraform and helm chart since we are moving to a different infra | [`545235c`](https://github.com/javaknight1/servicepro/commit/545235ce4322f943e241ffa9c4656f9a4e21c357) |
| 🔧 Miscellaneous | Added check for valid commit messages                                     | [`dceafa5`](https://github.com/javaknight1/servicepro/commit/dceafa55f27f64e818ba7e359508a547c8c91a13) |
| 🔧 Miscellaneous | Add release pipeline                                                      | [`c65d83d`](https://github.com/javaknight1/servicepro/commit/c65d83d36f27abb5c05753fe70d37995792f27f3) |
