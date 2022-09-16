all: person.png observation.png condition.png drugexposure.png procedure.png

person.png:
	layers compose --bundle graphmodel.bundle.yaml --type https://dartnet.info/graphModel/Person --output dot >person.dot
	dot person.dot -Tpng -o person.png
	rm person.dot

observation.png:
	layers compose --bundle graphmodel.bundle.yaml --type https://dartnet.info/graphModel/Observation --output dot >observation.dot
	dot observation.dot -Tpng -o observation.png
	rm observation.dot

condition.png:
	layers compose --bundle graphmodel.bundle.yaml --type https://dartnet.info/graphModel/Condition --output dot >condition.dot
	dot condition.dot -Tpng -o condition.png
	rm condition.dot


drugexposure.png:
	layers compose --bundle graphmodel.bundle.yaml --type https://dartnet.info/graphModel/DrugExposure --output dot >drugexposure.dot
	dot drugexposure.dot -Tpng -o drugexposure.png
	rm drugexposure.dot


procedure.png:
	layers compose --bundle graphmodel.bundle.yaml --type https://dartnet.info/graphModel/Procedure --output dot >procedure.dot
	dot procedure.dot -Tpng -o procedure.png
	rm procedure.dot



