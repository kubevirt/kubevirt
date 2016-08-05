PNGDIR=pngs/
UMLS=$(patsubst %,$(PNGDIR)/%,$(subst plantuml,png,$(wildcard *.plantuml)))

all: $(UMLS)
	echo $(UMLS)

$(PNGDIR)/%.png: %.plantuml
	plantuml -tpng $< -o $(PNGDIR)/
