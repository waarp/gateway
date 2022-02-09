##########
Pré-requis
##########

Système
=======

Les systèmes d'exploitation suivants sont officiellement supportés :

+-----------------------------+----------------+
| Système d'exploitation      | Architectures  |
+=============================+================+
| Linux 2.6.23 (glibc requis) | amd64, i386    |
+-----------------------------+----------------+
| Windows 7 ou Server 2008R2  | amd64, i386    |
+-----------------------------+----------------+

.. Cependant, la gateway étant écrite en langage *Go*, le système sur lequel elle
   sera installée doit faire parti des `systèmes supportés par le compilateur
   <https://golang.org/doc/install#requirements>`_.


Exécutables
===========

La *gateway* est composée de 2 exécutables:

``waarp-gatewayd``
   l'exécutable de la *gateway* elle-même. Cet exécutable
   est un serveur destiné à être exécuté en arrière-plan, typiquement via un
   gestionnaire de service (ex: ``systemd`` sous Linux).

``waarp-gateway``
  le client en ligne de commande permettant d'administrer
  la *gateway*. Ce client utilise l'interface REST de la *gateway* pour communiquer.
  Pour simplifier les commandes, il est recommander d'ajouter cet exécutable au
  ``$PATH`` du système. Un guide sur l'utilisation du client est disponible
  :any:`ici <user-guide-client>`.


Base de données
===============

Pour fonctionner, Waarp Gateway nécessite une base de donnée. Par défaut,
la *gateway* utilise une base embarquée SQLite stockée dans un fichier.
Dans ce cas de figure, aucune action n'est requise, au lancement de la Gateway,
le fichier base de données sera automatiquement créé.

Waarp Gateway supporte également les serveurs de base de données MySQL et
PostgreSQL. Pour utiliser ces serveurs comme base de données, les étapes
suivantes sont requises :

1) Créer une base de données vierge sur le serveur. Une base déjà existante
peut être utilisée, mais cela n'est pas recommandé.

2) Ajouter un utilisateur ayant le droit d'ajouter et de modifier des tables sur
la base de données en question. Cet utilisateur sera utilisé par la Gateway
pour s'authentifier auprès du serveur.


Les informations de connections à la base de données doivent ensuite être
renseignées dans le fichier de configuration de la *gateway* (cf.
:any:`configuration-file`). Une fois la base de données créée, elle sera ensuite
remplie automatiquement par la *gateway* elle-même.


Interface d'administration
==========================

Pour être administrée, la *gateway* inclue un serveur HTTP d'administration.
Par défaut, ce serveur écoute et répond en HTTP clair. Pour plus de sécurité,
il est recommandé de générer un certificat pour le serveur, et de l'ajouter
au fichier de configuration pour que les requêtes puissent être faites en
HTTPS au lieu de HTTP.


Fichier de configuration
========================

Pour fonctionner, la *gateway* nécessite un fichier de configuration en format
*.ini*. Ce fichier de configuration peut être généré avec la commande:

.. code-block:: shell

   waarp-gatewayd server -n -c chemin/de/la/configuration.ini


.. note::
   Bien qu'il soit possible d'utiliser la Gateway avec la configuration par
   défaut, il est fortement recommandé de consulter le détail du
   :any:`configuration-file` pour ensuite le modifier avec des valeurs plus
   adaptées à votre utilisation.
