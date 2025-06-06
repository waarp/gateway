##########
Pré-requis
##########

Système
=======

Les systèmes d'exploitation suivants sont officiellement supportés :

+-------------------------------------+----------------+
| Système d'exploitation              | Architecture   |
+=====================================+================+
| Linux kernel 3.2 minimum            | amd64          |
+-------------------------------------+----------------+
| Windows 10 (ou Server 2016) minimum | amd64          |
+-------------------------------------+----------------+

Exécutables
===========

Waarp Gateway est composée de 2 exécutables:

``waarp-gatewayd``
   l'exécutable de Gateway elle-même. Cet exécutable
   est un serveur destiné à être exécuté en arrière-plan, typiquement via un
   gestionnaire de service (ex: :program:`systemctl` sous Linux).

``waarp-gateway``
  le client en ligne de commande permettant d'administrer
  Gateway. Ce client utilise l'interface REST de Gateway pour communiquer.
  Pour simplifier les commandes, il est recommander d'ajouter cet exécutable au
  ``$PATH`` du système. Un guide sur l'utilisation du client est disponible
  :any:`ici <user-guide-client>`.


Base de données
===============

Pour fonctionner, Waarp Gateway nécessite une base de donnée. Par défaut,
Gateway utilise une base embarquée SQLite stockée dans un fichier.
Dans ce cas de figure, aucune action n'est requise, au lancement de Waarp Gateway,
le fichier base de données sera automatiquement créé.

Waarp Gateway supporte également les serveurs de base de données MySQL et
PostgreSQL. Pour utiliser ces serveurs comme base de données, les étapes
suivantes sont requises :

1) Créer une base de données vierge sur le serveur. Une base déjà existante
peut être utilisée, mais cela n'est pas recommandé.

2) Ajouter un utilisateur ayant le droit d'ajouter et de modifier des tables sur
la base de données en question. Cet utilisateur sera utilisé par Waarp Gateway
pour s'authentifier auprès du serveur.


Les informations de connections à la base de données doivent ensuite être
renseignées dans le fichier de configuration de Gateway (cf.
:any:`configuration-file`). Une fois la base de données créée, elle sera ensuite
remplie automatiquement par Gateway elle-même.


Interface d'administration
==========================

Pour être administrée, Gateway inclue un serveur HTTP d'administration.
Par défaut, ce serveur écoute et répond en HTTP clair. Pour plus de sécurité,
il est recommandé de générer un certificat pour le serveur, et de l'ajouter
au fichier de configuration pour que les requêtes puissent être faites en
HTTPS au lieu de HTTP.


Fichier de configuration
========================

Pour fonctionner, Gateway nécessite un fichier de configuration en format
*.ini*. Ce fichier de configuration peut être généré avec la commande:

.. code-block:: shell

   waarp-gatewayd server -n -c chemin/de/la/configuration.ini


.. note::
   Bien qu'il soit possible d'utiliser Waarp Gateway avec la configuration par
   défaut, il est fortement recommandé de consulter le détail du
   :any:`configuration-file` pour ensuite le modifier avec des valeurs plus
   adaptées à votre utilisation.
