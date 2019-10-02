************************
Paramètres de connection
************************

.. program:: waarp-gateway

.. option:: -a <ADDR>, --address=<ADDR>

   L'adresse de l'instance de gateway à interroger. Ce paramètre est requis.
   Cette adresse doit être fournie sous forme de DSN (Data Source Name)::

      <protocole>://<login>:<mot de passe>@<hôte>:<port>

   Le protocole peut être *http* ou *https* en fonction de la configuration de
   l'interface REST de la gateway.

   Le mot de passe est optionnel, et peut donc être omis. Si le mot de passe
   n'est pas renseigné dans la DSN, il sera demandé via un *prompt* du terminal.