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
- `-replay PATH`: 키보드 대신 리플레이 파일의 입력으로 진행하고, 입력이 끝나면 종료
- 조작 (임시): 좌우 화살표 또는 A·D, 점프는 Z 또는 Space
