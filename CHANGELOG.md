# Changelog

All notable changes to ServicePro will be documented in this file.

## [0.7.0] - 2026-02-07

### Release Features

| Type         | Description                                       | Commit                                                                                                 |
| ------------ | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ✨ Features  | Added graceful shutdown on backend startup        | [`3bf6431`](https://github.com/javaknight1/servicepro/commit/3bf6431851f8b5d794feff6ad3cd265d3df9a9aa) |
| ✨ Features  | Emit metrics on /metrics endpoint                 | [`ed6589a`](https://github.com/javaknight1/servicepro/commit/ed6589a56e16ae496efa06ad32a0a0210c854eec) |
| ✨ Features  | Upgraded logging logic                            | [`33520a9`](https://github.com/javaknight1/servicepro/commit/33520a94cb915b0ac932c3d3a04622142d40c20f) |
| ✨ Features  | Added calendar view for jobs                      | [`d93bf27`](https://github.com/javaknight1/servicepro/commit/d93bf27acbdac5a6c73ac5e86f268d07e4459852) |
| ✨ Features  | Created a/r aging report                          | [`97e380c`](https://github.com/javaknight1/servicepro/commit/97e380c0814e368b82e5bf405e2892005f2e7b1e) |
| 🐛 Bug Fixes | Added deep health checks for some of our services | [`c432ab4`](https://github.com/javaknight1/servicepro/commit/c432ab4b5697c3db49f70911d6b00b31ca0fe90a) |
| 🐛 Bug Fixes | Fixed token access on frontend                    | [`8cda579`](https://github.com/javaknight1/servicepro/commit/8cda57937662dd680271a712b4c18aabc91ec11e) |
| 🐛 Bug Fixes | Centralize any calls to fetch()                   | [`440abdb`](https://github.com/javaknight1/servicepro/commit/440abdb727fe4cda381becfcb24554f95f47a49a) |
| 🐛 Bug Fixes | Added csp/hsts headers for frontend               | [`f262625`](https://github.com/javaknight1/servicepro/commit/f262625de4f3823e4becf8e84a1b753d8e489046) |
| 🐛 Bug Fixes | Fixed error handling for Sentry                   | [`e0d9a4f`](https://github.com/javaknight1/servicepro/commit/e0d9a4f759ca95e46000eba56f1aee08a918ff4f) |
| 🐛 Bug Fixes | Remove any usage of any type in frontend          | [`d9749b2`](https://github.com/javaknight1/servicepro/commit/d9749b238418b91b55d94e5184babded93e127a5) |
| 🐛 Bug Fixes | Remoed ts-nocheck                                 | [`dddd57a`](https://github.com/javaknight1/servicepro/commit/dddd57a69749a0466190eebdd20a3ab2030f59d4) |
| 🐛 Bug Fixes | Enabled noUnusedLocals and noUnusedParameters     | [`1f414c8`](https://github.com/javaknight1/servicepro/commit/1f414c86cdf647590569cba08044e8794eb05692) |

### Dev Features

| Type             | Description                                              | Commit                                                                                                 |
| ---------------- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ♻️ Refactoring   | Cleaned up the frontend and removed unused code          | [`b6fc6d4`](https://github.com/javaknight1/servicepro/commit/b6fc6d4ba5b4ece00714c1e07eb915c4c85885c0) |
| ♻️ Refactoring   | Created file download library to be used across frontend | [`9de879d`](https://github.com/javaknight1/servicepro/commit/9de879d0397d1e6bf5392b69578778565911d66b) |
| ♻️ Refactoring   | Centralize url search params lib                         | [`a98c98e`](https://github.com/javaknight1/servicepro/commit/a98c98e97211cd42bd22c80d7e52de0cd8016e8d) |
| ♻️ Refactoring   | Consolidate cache hooks                                  | [`86f3ae4`](https://github.com/javaknight1/servicepro/commit/86f3ae4454be1d1b136455fa01552831e93cf4a4) |
| ✅ Tests         | Removed unit tests not being used anymore                | [`8d27820`](https://github.com/javaknight1/servicepro/commit/8d278200a13e0d56854f2a11f488ec7ba8dcc129) |
| ✅ Tests         | Updated frontend with over 70% code coverage             | [`7e75693`](https://github.com/javaknight1/servicepro/commit/7e756930eda30530af5ba248b3f39d4fabb38834) |
| ✅ Tests         | Updated integration tests for frontend                   | [`f7e0b09`](https://github.com/javaknight1/servicepro/commit/f7e0b099e9224f3a029976e2c38c54b3406a9951) |
| ✅ Tests         | Integrated sql query performance metrics                 | [`f059c9f`](https://github.com/javaknight1/servicepro/commit/f059c9ff033e7c47ee824d0eba95b8be2ace796d) |
| 📚 Documentation | Updated TODO from items from MVP doc                     | [`1dfa803`](https://github.com/javaknight1/servicepro/commit/1dfa803b19fb84641c1be77010c870d8fcb55b46) |
| 📚 Documentation | Completed openapi swagger docs for API                   | [`81ec841`](https://github.com/javaknight1/servicepro/commit/81ec841b438e4e364786cb3c6823460c549c39f1) |
| 🚀 CI/CD         | Added jobs to check for dead code                        | [`416b86b`](https://github.com/javaknight1/servicepro/commit/416b86b3d57b9b3006e685312d8dae6be65344f1) |
| 🚀 CI/CD         | Frontend analysis and ptimization                        | [`948583d`](https://github.com/javaknight1/servicepro/commit/948583d830c54ed8afe8461e5044b84357f6f507) |
| 🚀 CI/CD         | Bundle costs for pushes, commits, and release            | [`5531cbb`](https://github.com/javaknight1/servicepro/commit/5531cbbca7be96fb74229249e71a2cbcef3692e0) |

## [0.6.0] - 2026-02-01

### Release Features

| Type             | Description                                                     | Commit                                                                                                 |
| ---------------- | --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ✨ Features      | Jobs have a status flow                                         | [`11023e6`](https://github.com/javaknight1/servicepro/commit/11023e62c124596b7220809f87ad7a89df8f5c2a) |
| ✨ Features      | Download quote and invoice pdfs from dashboard                  | [`0bad53e`](https://github.com/javaknight1/servicepro/commit/0bad53e4843f942cf446993e1068f3d4bfa6f27a) |
| ✨ Features      | Created multiple statuses for quotes                            | [`b3728c0`](https://github.com/javaknight1/servicepro/commit/b3728c05649d76d3bc4513901889fb4bfdea1a64) |
| ✨ Features      | Create pkg to send SMS                                          | [`ed154af`](https://github.com/javaknight1/servicepro/commit/ed154af821456262c9ce02bbba5b0d94aede373a) |
| 🎯 Minor Changes | Added new fields to customer to set contact preference          | [`b533bc5`](https://github.com/javaknight1/servicepro/commit/b533bc546903795f509ab1a35de0c937e443b4b1) |
| 🐛 Bug Fixes     | Fixed issue with Row Menu Item getting cut off                  | [`5205c72`](https://github.com/javaknight1/servicepro/commit/5205c7223c4e14885549f82b095be6cfcdb281df) |
| 🐛 Bug Fixes     | Enforce random secret JWT key                                   | [`37f8bf4`](https://github.com/javaknight1/servicepro/commit/37f8bf4973eab32b97529eef8522b0b2a0c68290) |
| 🐛 Bug Fixes     | Moved JWT token storage from localstorage to httpOnly cookies   | [`ea4f9d7`](https://github.com/javaknight1/servicepro/commit/ea4f9d72af2f8a40943c10391fc550de181a89c8) |
| 🐛 Bug Fixes     | Bump @hookform/resolvers from 3.10.0 to 5.2.2 in /frontend      | [`56bcdcb`](https://github.com/javaknight1/servicepro/commit/56bcdcb7d48547e0b41e33e6fd9c00e8a5d3b275) |
| 🐛 Bug Fixes     | Bump softprops/action-gh-release from 1 to 2                    | [`3fda576`](https://github.com/javaknight1/servicepro/commit/3fda576db5741d7245580adc185ee4ffb25aec2c) |
| 🐛 Bug Fixes     | Bump actions/setup-python from 5 to 6                           | [`ca22afd`](https://github.com/javaknight1/servicepro/commit/ca22afda850a4238518aa33bcbbf5888f53d0db2) |
| 🐛 Bug Fixes     | Bump the go-minor-patch group in /backend with 13 updates       | [`a8fcfbc`](https://github.com/javaknight1/servicepro/commit/a8fcfbc4358600cee04184c05742990905b3f766) |
| 🐛 Bug Fixes     | Bump eslint-plugin-react-hooks from 4.6.2 to 7.0.1 in /frontend | [`6decc03`](https://github.com/javaknight1/servicepro/commit/6decc03ca29b6e8789bf15f9225f95a4549abb9d) |
| 🐛 Bug Fixes     | Bump actions/checkout from 4 to 6                               | [`4eb4b53`](https://github.com/javaknight1/servicepro/commit/4eb4b5317a2ceb65d971e2cbf01df6ecf5fc6ab8) |
| 🐛 Bug Fixes     | Bump tailwindcss from 3.4.18 to 4.1.18 in /frontend             | [`aaf5b3c`](https://github.com/javaknight1/servicepro/commit/aaf5b3cf23b67d15ea0d991a8ba281bfa7922cd0) |
| 🐛 Bug Fixes     | Bump vite from 5.4.21 to 7.3.1 in /frontend                     | [`2312ce8`](https://github.com/javaknight1/servicepro/commit/2312ce81476cd790587ea9d2d509791071fbe664) |
| 🐛 Bug Fixes     | Bump react-dom and @types/react-dom in /frontend                | [`1fc7cf5`](https://github.com/javaknight1/servicepro/commit/1fc7cf51830db97ca52433076083e648641326bf) |
| 🐛 Bug Fixes     | Bump the npm-minor-patch group in /frontend with 19 updates     | [`969bebe`](https://github.com/javaknight1/servicepro/commit/969bebee126833956667a752d626c828beda194f) |
| 🐛 Bug Fixes     | Bump react and @types/react in /frontend                        | [`35b2a27`](https://github.com/javaknight1/servicepro/commit/35b2a274d23139f9f879cd41aaf84718d94d0763) |
| 🐛 Bug Fixes     | Bump @headlessui/react from 1.7.19 to 2.2.9 in /frontend        | [`5748ec0`](https://github.com/javaknight1/servicepro/commit/5748ec03d9007111ee1f096bb67242ca42ecd59a) |
| 🐛 Bug Fixes     | Bump actions/setup-node from 4 to 6                             | [`bb36ca9`](https://github.com/javaknight1/servicepro/commit/bb36ca97b2d7d2bf48bdda4c81196d7f9f862567) |
| 🐛 Bug Fixes     | Bump react-router-dom from 6.30.2 to 7.13.0 in /frontend        | [`360228c`](https://github.com/javaknight1/servicepro/commit/360228cff8d394f1a7cb283c242686729d7a3d5a) |
| 🐛 Bug Fixes     | Bump rollup-plugin-visualizer from 5.14.0 to 6.0.5 in /frontend | [`299988b`](https://github.com/javaknight1/servicepro/commit/299988bf3e24f4187309428474a4fe5597c66507) |
| 🐛 Bug Fixes     | Bump @vitejs/plugin-react from 4.7.0 to 5.1.2 in /frontend      | [`5b4ce8e`](https://github.com/javaknight1/servicepro/commit/5b4ce8e5d9fb9a5dc71e3a65bbd1147547664bda) |
| 🐛 Bug Fixes     | Updated packages                                                | [`82a6f50`](https://github.com/javaknight1/servicepro/commit/82a6f50b1cf2df3593200b51eda163081be29099) |
| 🐛 Bug Fixes     | Fixed linting with invalud package-lock.json                    | [`caddf2a`](https://github.com/javaknight1/servicepro/commit/caddf2a2579ae15601b6b1ee744e0cc851a3e9d1) |
| 🐛 Bug Fixes     | Bump zod from 3.25.76 to 4.3.6 in /frontend                     | [`1eefc69`](https://github.com/javaknight1/servicepro/commit/1eefc697fb6261506681c977e48487fe13f46817) |
| 🐛 Bug Fixes     | Bump vite-plugin-pwa from 0.21.2 to 1.2.0 in /frontend          | [`144a1a5`](https://github.com/javaknight1/servicepro/commit/144a1a5b9987dff04ca8f99f9f77bd3e63cda196) |
| 🐛 Bug Fixes     | Bump zustand from 4.5.7 to 5.0.10 in /frontend                  | [`20c5c3e`](https://github.com/javaknight1/servicepro/commit/20c5c3e56920ee7a202dc2f3f5182500ca6cd5af) |
| 🐛 Bug Fixes     | Bump actions/cache from 4 to 5                                  | [`2fdfdee`](https://github.com/javaknight1/servicepro/commit/2fdfdee629ba8c16caf29fc547ccc9ead3c15ad3) |
| 🐛 Bug Fixes     | Fixed linting with invalud package-lock.json                    | [`0643eed`](https://github.com/javaknight1/servicepro/commit/0643eedf08465096ee01dee592eb67394b54a59a) |
| 🐛 Bug Fixes     | Fixed linting and vulnerabilities from audit                    | [`f8b549b`](https://github.com/javaknight1/servicepro/commit/f8b549bf105d1edf7e20faa3e3c66ccdb6c2fec9) |

### Dev Features

| Type             | Description                                                        | Commit                                                                                                 |
| ---------------- | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| ✅ Tests         | Seed database for development                                      | [`9b59b88`](https://github.com/javaknight1/servicepro/commit/9b59b8886d61c7b3908d135bc815d0f886e44f35) |
| ✅ Tests         | Added dependabot that would check if any dependencies are outdated | [`feecf1c`](https://github.com/javaknight1/servicepro/commit/feecf1cd3b06f4ad39adc71407fdc3d0763600b5) |
| ✅ Tests         | Fixed the failing unit tests for PDFs and emailing                 | [`00bc2b1`](https://github.com/javaknight1/servicepro/commit/00bc2b13a801164ba107f58c439bab8c1836b89b) |
| 📚 Documentation | Updated TODO with new tasks completed                              | [`0a8bc24`](https://github.com/javaknight1/servicepro/commit/0a8bc24d4547ac4357bb85a94d0ed0a722eb8481) |

## [0.5.0] - 2026-01-29

### Release Features

| Type             | Description                                                                     | Commit                                                                                                 |
| ---------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ✨ Features      | Added ability to add users to jobs                                              | [`3a56f9e`](https://github.com/javaknight1/servicepro/commit/3a56f9e69b17538d542979c5d2658c887f72a0d3) |
| ✨ Features      | Create entire pipeline for creating and sending invoices and accepting payments | [`b76480c`](https://github.com/javaknight1/servicepro/commit/b76480c37a65e46623214586e7dee13cae447e61) |
| 🎯 Minor Changes | Created pages with links in footer                                              | [`0d7218c`](https://github.com/javaknight1/servicepro/commit/0d7218c96c8945a6d78f0e143f4b98d344b30db1) |

### Dev Features

| Type     | Description                                | Commit                                                                                                 |
| -------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| ✅ Tests | Fixed some backend unit tests for invoices | [`7640b6a`](https://github.com/javaknight1/servicepro/commit/7640b6a4fcab5c6a572053e22b99204884b6bf86) |

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
