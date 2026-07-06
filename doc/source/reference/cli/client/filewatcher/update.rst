==========================
Modifier un filewatcher
==========================

.. program:: waarp-gateway filewatcher update

Remplace les attributs du *filewatcher* demandé avec ceux donnés. Les attributs
omis restent inchangés.

**Commande**

.. code-block:: shell

   waarp-gateway filewatcher update "<FLOW>"

**Options**

.. option:: -f <FLOW>, --flow=<FLOW>

   Le nouveau nom du flux. Doit être unique.

.. option:: -i <INTERVAL>, --interval=<INTERVAL>

   La nouvelle fréquence d'interrogation du partenaire distant.
   Les unités de temps acceptées sont : ``h`` (heures), ``m`` (minutes) et
   ``s`` (secondes). Plusieurs unités peuvent être combinées (ex: ``1h15m30s``).

.. option:: -p <PATTERN>, --pattern=<PATTERN>

   Le nouveau motif de fichier à surveiller, au format
   `glob <https://en.wikipedia.org/wiki/Glob_(programming)>`_ (ex: ``*.txt``).

.. option:: --partner=<PARTNER>

   Le nouveau nom du partenaire distant à interroger.

.. option:: -a <ACCOUNT>, --account=<ACCOUNT>

   Le nouvel identifiant du compte distant.

.. option:: -c <CLIENT>, --client=<CLIENT>

   Le nouveau nom du client local à utiliser.

.. option:: -r <RULE>, --rule=<RULE>

   Le nouveau nom de la règle de réception à utiliser.

.. option:: --disabled

   Active ou désactive le *filewatcher* au démarrage.

.. option:: --no-duplicate-check

   Active ou désactive la détection de fichiers en double.

|

**Exemple**

.. code-block:: shell

   waarp-gateway filewatcher update 'my-filewatcher' --flow 'my-filewatcher-2' --interval '10m' --pattern '*.csv'
