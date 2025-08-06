.. _ref-cli-transfer-preregister:

===================================
Préenregistrer un transfert serveur
===================================

.. program:: waarp-gateway transfer preregister

Préenregistre un nouveau transfert serveur avec les attributs ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway transfer preregister

**Options**

.. option:: -f <FILENAME>, --file=<FILENAME>

   Le chemin du fichier à transférer. Si le chemin est relatif, il sera relatif
   au dossier de la règle (ou du serveur en l'absence de dossier de règle).

.. option:: -r <RULE>, --rule=<RULE>

   Le nom de la règle utilisée pour le transfert.

.. option:: -w <DIRECTION>, --way=<DIRECTION>

   La direction du transfert. Peut être ``send`` ou ``receive``.

.. option:: -s <SERVER>, --server=<SERVER>

   Le nom du serveur local sur lequel le transfert sera effectué.

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le nom du compte local utilisé par le partenaire duquel la requête de transfert
   sera émise.

.. option:: -d <DATE>, --due-date=<DATE>

   La date limite du transfert en format ISO 8601. Une fois cette date dépassée,
   si le transfert n'a pas commencé, il tombera en erreur.

.. option:: -i <KEY:VAL>, --info=<KEY:VAL>

   Une liste d'informations personnalisées à attacher au transfert. Les informations
   prennent la forme d'une liste de paires clé:valeur. Répéter l'option pour ajouter
   des paires supplémentaires.

**Exemple**

.. code-block:: shell

   waarp-gateway transfer preregister -f 'path/to/file' -r 'règle_1' -w 'send' -s 'sftp_server' -l 'toto' -d '2026-01-01T01:00:00Z'
