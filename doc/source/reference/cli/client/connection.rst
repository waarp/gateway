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

.. option:: -i, --insecure

   Désactive la validation du certificat de l'interface REST du service Gateway.
   Peut être utilisé pour les certificats autosignés et les tests.

   .. warning::

      Comme la validation du certificat serveur n'est plus faite, le client fait
      aveuglément confiance au serveur.

      Cela peut représenter une faille de sécurité si vous n'êtes pas absolument
      certain du serveur quand vous utilisez cette option.


.. envvar:: WAARP_GATEWAY_ADDRESS

   Si l'adresse de la Gateway n'est pas renseignée dans la commande via l'option
   `-a`, l'adresse sera récupérée dans cette variable d'environnement. La syntaxe
   de l'adresse reste identique à celle décrite ci-dessus.

.. envvar:: WAARP_GATEWAY_INSECURE

   Désactive la validation du certificat de l'interface REST du service Gateway.
   (équivalent à l'option :option:`-i`)
