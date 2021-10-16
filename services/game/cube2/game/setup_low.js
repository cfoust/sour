Module.setPlayerModels = function() {
  BananaBread.setPlayerModelInfo("snoutx10k", "snoutx10k", "snoutx10k", "snoutx10k/hudguns", 0, 0, 0, 0, 0, "snoutx10k", "snoutx10k", "snoutx10k", true);
};

Module.tweakDetail = function() {
  BananaBread.execute('fog 10000'); // disable fog
  BananaBread.execute('maxdebris 10');
  if (Module.benchmark) {
    BananaBread.execute('shaderdetail 1');
    BananaBread.execute('maxdynlights 0');
  }
};

Module.loadDefaultMap = function() {
};
