====================
Ajouter un transfert
====================

.. program:: waarp-gateway transfer add

Programme un nouveau transfert avec les attributs ci-dessous.

.. option:: -f <FILENAME>, --file=<FILENAME>

   Le chemin du fichier à transférer. Si le chemin est relatif, il sera relatif
   au dossier de la règle (ou du serveur en l'absence de dossier de règle).

.. option:: -o <FILENAME>, --out=<FILENAME>

   Le chemin de destination du fichier transféré. Si le chemin est relatif, il
   sera relatif au dossier de la règle (ou du serveur en l'absence de dossier
   de règle).

.. option:: -w <DIRECTION>, --way=<DIRECTION>

   La direction du transfert. Peut être ``send`` ou ``receive``.

.. option:: -c <CLIENT>, --client=<CLIENT>

   Le nom du client local avec lequel le transfert va être exécuté. Peut être
   omit si la gateway ne possède qu'un seul client du protocole concerné, auquel
   cas, le client en question sera sélectionné automatiquement.

.. option:: -p <PARTNER>, --partner=<PARTNER>

   Le nom du partenaire distant avec lequel le transfert va être effectué.

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le nom du compte distant utilisé par Gateway pour d'identifier
   auprès du partenaire de transfert.

.. option:: -r <RULE>, --rule=<RULE>

   Le nom de la règle utilisée pour le transfert.

.. option:: -d <DATE>, --date=<DATE>

   La date de début du transfert en format ISO 8601. Par défaut, le transfert
   débutera immédiatement.

.. option:: -n <FILENAME>, --name=<FILENAME>

   .. deprecated:: 0.5.0

      Remplacé par `--out`.

   Le nom du fichier après le transfert. Par défaut, le nom d'origine est
   utilisé.

.. option:: -i <KEY:VAL>, --info=<KEY:VAL>

   Une liste d'informations personnalisées à attacher au transfert. Les informations
   prennent la forme d'une liste de paires clé:valeur. Répéter l'option pour ajouter
   des paires supplémentaires.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' transfer add -f 'path/to/file' -w 'send' -p 'waarp_sftp' -l 'toto' -r 'règle_1' -d '2022-01-01T01:00:00Z'
