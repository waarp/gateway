############
Introduction
############


L'objectif de cette section est de proposer un tour d'horizon des
fonctionnalités de Waarp Gateway et de vous guider dans la mise en place d'une
instance et sa configuration pour mettre en place :

- des transferts SFTP & R66 simples, en réception et en envoi respectivement
- un rebond SFTP -> R66
- la mise en place d'une instance en grappe

Il est recommandé de suivre ce guide dans une machine virtuelle de test prévue à
cet effet.

Dans la suite, les commandes sont données pour une installation faite sous
Centos 8 avec les packages RPM. Elles devront être adaptées pour d'autres
systèmes.

Ouvrez un shell dans la VM de test et suivez la :any:`procédure d'installation
<install>` pour commencer.

Convention
==========

Dans la suite de ce guide, les commandes nécessaires à la mise en place sont
indiquées avec la sortie qu'elles produisent à titre d'exemple. Cette sortie
peut varier selon l'environnement.

Les commandes précédées du signe dièse (``#``) doivent être exécutées en tant
qu'administrateur (``root``) et les commandes précédées du signe dollar (``$``)
peuvent être exécutées en tant qu'utilisateur normal.
