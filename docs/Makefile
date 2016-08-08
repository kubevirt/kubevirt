PNGDIR=pngs/
UMLS=$(wildcard *.plantuml)
PNGS=$(patsubst %,$(PNGDIR)/%,$(subst plantuml,png,$(UMLS)))

all: $(UMLS)

continuous:
	while sleep 0.2 ; do make VIEW=1 $$(inotifywait --format %w $(UMLS) | sed -e "s/^/pngs\// ; s/plantuml/png/" ; sleep 1) ; done

$(PNGDIR)%.png: %.plantuml
	plantuml -tpng $< -o $(PNGDIR)/
	[[ -n "$(VIEW)" ]] && eog $@ &
