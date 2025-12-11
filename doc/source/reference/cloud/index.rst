.. _reference-cloud:

###############
Instances cloud
###############

Au lieu d'utiliser le disque local pour le transfert de fichiers, la gateway est
également capable de stocker ou récupérer les fichiers sur une instance cloud
distante (voir la section :any:`gestion des dossiers <instances-cloud>`)
pour plus de détails sur leur utilisation).

Il est à noter que Waarp Gateway utilise un cache sur le disque local pour
palier aux limitations des instances cloud. Le dossier de cache est configurable
dans :ref:`le fichier de configuration <configuration-file>`. Le reste des
options du cache sont configurables instance par instance via les "options"
de l'instance.

Cette liste regroupe les types d'instances cloud supportés par la gateway, ainsi
que leur configurations respectives.

.. toctree::
   :maxdepth: 1

   s3
   azurefiles
   azureblob
   gcs
