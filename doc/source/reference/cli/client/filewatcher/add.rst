========================
Ajouter un filewatcher
========================

.. program:: waarp-gateway filewatcher add

Ajoute un nouveau *filewatcher* avec les attributs renseignés.

**Commande**

.. code-block:: shell

   waarp-gateway filewatcher add

**Options**

.. option:: -f <FLOW>, --flow=<FLOW>

   Le nom du flux auquel le *filewatcher* appartient. Doit être unique.

.. option:: -i <INTERVAL>, --interval=<INTERVAL>

   La fréquence à laquelle le *filewatcher* interrogera le partenaire distant.
   Les unités de temps acceptées sont : ``h`` (heures), ``m`` (minutes) et
   ``s`` (secondes). Plusieurs unités peuvent être combinées (ex: ``1h15m30s``).

.. option:: -p <PATTERN>, --pattern=<PATTERN>

   Le motif de fichier à surveiller, au format
   `glob <https://en.wikipedia.org/wiki/Glob_(programming)>`_ (ex: ``*.txt``).

.. option:: --partner=<PARTNER>

   Le nom du partenaire distant à interroger.

.. option:: -a <ACCOUNT>, --account=<ACCOUNT>

   L'identifiant du compte distant à utiliser pour l'authentification.

.. option:: -c <CLIENT>, --client=<CLIENT>

   Le nom du client local à utiliser pour la connexion.

.. option:: -r <RULE>, --rule=<RULE>

   Le nom de la règle de réception à utiliser pour les transferts.

.. option:: --disabled

   Crée le *filewatcher* dans un état désactivé au démarrage.

.. option:: --no-duplicate-check

   Désactive la détection de fichiers en double. Par défaut, le *filewatcher*
   ignore les fichiers déjà récupérés lors d'un passage précédent.

|

**Exemple**

.. code-block:: shell

   waarp-gateway filewatcher add --flow 'my-filewatcher' --interval '5m' --pattern '*.txt' --partner 'my-partner' --account 'my-account' --client 'my-client' --rule 'my-rule'
