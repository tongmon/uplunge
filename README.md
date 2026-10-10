# uplunge
Advanced to the top game

- [기획서](docs/design.md)
- [로드맵](docs/roadmap.md)
- [레퍼런스 분석 노트](docs/references.md)

## 실행

```
go run ./cmd/uplunge
```

- `-scale N`: 창 배율 (기본 2)
- `-tuning PATH`: 조정값 파일 (기본 `data/tuning.json`)
- `-chunks PATH`: LDtk 조각 파일 (기본 `assets/chunks/chunks.ldtk`)
- `-chunk NAME`: 플레이할 조각 이름 (기본: 리플레이의 조각, 없으면 첫 번째 조각)
- `-record PATH`: 종료할 때 이번 판의 입력을 리플레이 파일로 저장
- `-replay PATH`: 키보드 대신 리플레이 파일의 입력으로 진행하고, 입력이 끝나면 종료. 녹화 때와 조정값이나 조각 내용이 다르면 경고를 출력함
- `-shots 0,30,120`: 지정한 틱의 화면을 PNG로 저장하고, 마지막 틱 뒤에 종료 (`-replay`와 함께 쓰면 매번 같은 화면)
- `-shots-dir DIR`: PNG 저장 폴더 (기본 `out/shots`)
- `-reload`: 실행 중 `-tuning` 파일이 바뀌면 다시 읽음 (0.5초마다 내용을 비교). 판정 박스 크기는 발 가운데를 기준으로 바뀌고, 벽이나 천장과 겹치면 크기만 이전 값으로 두었다가 공간이 생기면 적용. `-record`와는 함께 쓸 수 없음
- 조작 (임시): 좌우 화살표 또는 A·D, 점프는 Z 또는 Space
