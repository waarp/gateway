.. _user-webui-dashboard:

################
Tableau de bord
################

Section *Supervision*, adresse ``/webui/dashboard``. Aucun droit particulier
n'est requis pour la page : tout utilisateur authentifié voit les services du
nœud. Les blocs sur les transferts demandent le droit de lecture des
transferts ; leurs actions demandent le droit d'écriture.

L'écran répond en un coup d'œil aux trois questions d'un responsable de
production : tout tourne-t-il, quelque chose demande-t-il une action,
l'activité est-elle normale. Rien n'est nouveau dans les données : la page
assemble ce que les écrans :doc:`etat-des-services` et
:doc:`suivi-des-transferts` montrent chacun de leur côté, et chaque chiffre
mène à l'écran qui le détaille, filtré. Un chiffre du tableau de bord est
toujours celui de l'écran qu'il ouvre.

.. list-table::
   :header-rows: 1
   :widths: 25 50 25

   * - Bloc
     - Contenu
     - Mène à
   * - Services du nœud
     - Trois tuiles, celles de l'état des services : services système,
       serveurs locaux, clients locaux, avec le nombre de services en ligne
       sur le total et ce qui ne va pas le cas échéant.
     - L'état des services, sur le groupe choisi.
   * - Transferts en cours
     - Le nombre de transferts en cours ; en pied, les transferts planifiés,
       en pause, interrompus et attendus.
     - Le suivi des transferts filtré sur les transferts en cours.
   * - Transferts en erreur
     - Le nombre de transferts encore en erreur, donc à traiter ; en pied,
       combien ont démarré depuis une heure.
     - Le suivi des transferts filtré sur les transferts en erreur.
   * - Terminés sur 24 h
     - Le nombre de transferts terminés sur les 24 dernières heures ; en
       pied, le volume transféré et les transferts annulés.
     - Le suivi des transferts filtré sur les transferts terminés depuis le
       début de la période.
   * - Activité des 24 dernières heures
     - Un graphique en barres, une par heure de début : les transferts
       terminés en plein, les transferts encore en erreur en hachuré avec leur
       nombre au-dessus. La section *Valeurs par heure*, repliée, donne les
       valeurs exactes ; son titre porte les totaux.
     -
   * - Dernières erreurs
     - Les cinq transferts en erreur les plus récents : heure de début,
       fichier (un lien vers la fiche du transfert), règle et partenaire,
       cause et détail de l'erreur, menu d'actions.
     - La fiche du transfert ; le suivi filtré sur les transferts en erreur.


Définitions
===========

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Terme
     - Sens
   * - Sur 24 h
     - Les transferts **démarrés** dans les 24 dernières heures : les
       23 heures pleines qui précèdent l'heure en cours, et l'heure en cours.
       La période se mesure à l'heure de début du transfert.
   * - En erreur
     - Les transferts encore en erreur, qui attendent une reprise, une
       relance ou une annulation. Un transfert repris ou relancé avec succès
       quitte ce compte ; un transfert annulé passe à l'historique, au statut
       *Annulé*. Le graphique ne montre donc que les transferts encore en
       erreur, par heure de début : une erreur reprise, relancée ou annulée
       depuis n'y figure plus.
   * - Terminés, annulés
     - Les transferts de l'historique démarrés dans la période, au statut
       *Terminé* ou *Annulé*.
   * - Volume transféré
     - La somme des tailles des transferts terminés de la période.


Actions sur une erreur
======================

Le menu « ⋯ » d'une ligne des dernières erreurs propose les actions que
l'état du transfert permet, les mêmes que dans le :doc:`suivi des transferts
<suivi-des-transferts>` : *Voir le transfert*, *Reprendre*, *Relancer*, puis
*Annuler le transfert*. Un transfert serveur ne se reprend ni ne se relance
depuis la Gateway : il reprend à l'initiative du partenaire. Une action
demandée depuis le tableau de bord y revient, avec sa notification.


Rafraîchissement
================

La page actualise ses chiffres toutes les 30 secondes sans se recharger, et
sur le bouton *Actualiser* ; l'heure de la dernière lecture est affichée dans
l'en-tête. Les chiffres sont lus une fois par période quel que soit le nombre
de navigateurs qui affichent la page.

Quand la base de données ne répond plus, la page garde ses derniers chiffres
et le dit : un encadré indique depuis quand la base ne répond pas et de quand
datent les chiffres affichés. Les tuiles des services, lues en mémoire,
restent exactes.

Sans JavaScript, le bouton *Actualiser* recharge la page ; la section des
valeurs et les menus d'actions fonctionnent tels quels.


Nœud et grappe
==============

Les services sont ceux du nœud qui sert la page, nommé dans le titre de leur
rangée. Les transferts sont ceux de toute la grappe, dont la base de données
est partagée.
