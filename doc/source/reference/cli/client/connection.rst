#######################
Paramètres de connexion
#######################

.. program:: waarp-gateway

.. describe:: waarp-gateway

.. option:: -a <ADDRESS>, --address=<ADDRESS>

   L'adresse de l'instance de gateway à interroger. Si le paramètre est absent,
   l'adresse sera récupérée depuis la variable d'environnement
   :envvar:`WAARP_GATEWAY_ADDRESS` (voir ci-dessous).
   Cette adresse doit être fournie sous forme de DSN (Data Source Name):

      [http|https]://<login>:<mot de passe>@<hôte>:<port>`

   Le protocole peut être *http* ou *https* en fonction de la configuration de
   l'interface REST de la gateway.

   Le login et le mot de passe requis sont les identifiants d'un
   :any:`utilisateur <reference-cli-client-users>`. L'utilisateur et le mot de passe peuvent
   être omis, au quel cas, il seront demandés via un *prompt* du terminal.

.. envvar:: WAARP_GATEWAY_ADDRESS

   Si l'adresse de la Gateway n'est pas renseignée dans la commande via l'option
   `-a`, l'adresse sera récupérée dans cette variable d'environnement. La syntaxe
   de l'adresse reste identique à celle décrite ci-dessus.
