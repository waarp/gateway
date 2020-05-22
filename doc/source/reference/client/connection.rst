########################
Paramètres de connection
########################

.. program:: waarp-gateway

.. describe:: waarp-gateway <ADDR>

.. option:: <ADDR>

   L'adresse de l'instance de gateway à interroger. Ce paramètre est requis.
   Cette adresse doit être fournie sous forme de DSN (Data Source Name)::

      http://<login>:<mot de passe>@<hôte>:<port>`

   Le protocole peut être *http* ou *https* en fonction de la configuration de
   l'interface REST de la gateway.

   Le login et le mot de passe requis sont les identifiants d'un
   :doc:`utilisateur <user/index>`. Le mot de passe peut être omis, au quel cas,
   il sera demandé via un *prompt* du terminal.