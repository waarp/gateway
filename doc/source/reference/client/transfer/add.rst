====================
Ajouter un transfert
====================

.. program:: waarp-gateway transfer add

.. describe:: waarp-gateway <ADDR> transfer add

Programme un nouveau transfert avec les attributs ci-dessous.

.. option:: -f <FILENAME>, --file=<FILENAME>

   Le nom du fichier à transférer.

.. option:: -n <FILENAME>, --name=<FILENAME>

   Le nom du fichier après le transfer. Par défaut, le nom d'origine est
   utilisé.

.. option:: -w <DIRECTION>, --way=<DIRECTION>

   La direction du transfer. Peut être ``pull`` ou ``push``.

.. option:: -p <PARTNER>, --partner=<PARTNER>

   Le nom du partenaire distant avec lequel le transfert va être effectué.

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le nom du compte distant utilisé par la gateway pour d'identifier
   auprès du partenaire de transfert.

.. option:: -r <RULE>, --rule=<RULE>

   Le nom de la règle utilisée pour le transfert.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 transfer add -f path/to/file -w push -p waarp_sftp a toto -r règle_1