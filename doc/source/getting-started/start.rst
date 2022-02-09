#########
Démarrage
#########


La configuration par défaut installée par les packages est suffisante pour
lancer Waarp Gateway.

Celle-ci peut être trouvée dans le fichier
:file:`/etc/waarp-gateway/waarp-gateway.ini`.

.. seealso::


La documentation complète du fichier de configuration est disponible
:any:`ici <configuration-file>`.
Après l'installation, la Gateway n'est pas démarrée. Lançons le service avec la
commande :

.. code-block:: bash

   systemctl start waarp-gatewayd

Vous pouvez vérifier que le service a bien démarré avec la commande

.. code-block:: bash

   systemctl status waarp-gatewayd

.. seealso::

   Pour plus d'information sur la gestion du service, ou savoir comment procéder
   si l'installation n'a pas été faite avec les dépôts ou les packages,
   consultez la page :any:`gestion du service <service_management>`.

Un autre moyen de savoir si le service est bien démarré est d'utiliser le client
en ligne de commande (nommé après "le client").

Le serveur expose une API REST, et le client en ligne de commande
:program:`waarp-gateway` est le moyen
recommandé d'interagir avec le serveur pour le gérer et l'administrer.

Toutes les commandes du client acceptent l'adresse de l'interface REST du
serveur comme premier argument.

Par exemple, la commande suivante permet de consulter le statut du serveur que
nous venons de lancer :

.. code-block:: shell-session

   # waarp-gateway -a "http://admin:admin_password@127.0.0.1:8080" status
   Waarp-Gateway services:
   [Active]  Admin
   [Active]  Controller
   [Active]  Database

L'adresse du serveur peut également être renseignée par une variable
d'environnement. Pour simplifier les exemples dans la suite, nous allons définir
la variable d'environnement :envvar:`WAARP_GATEWAY_ADDRESS` :

.. code-block:: shell-session

   # export WAARP_GATEWAY_ADDRESS="http://admin:admin_password@localhost:8080"
   # waarp-gateway status
   Waarp-Gateway services:
   [Active]  Admin
   [Active]  Controller
   [Active]  Database
